-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Functions table
CREATE TABLE functions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    runtime VARCHAR(50) NOT NULL,
    handler VARCHAR(255) NOT NULL,
    code_source TEXT NOT NULL,
    code_source_type VARCHAR(20) NOT NULL DEFAULT 'inline',
    code_checksum VARCHAR(64) NOT NULL,
    code_size BIGINT NOT NULL,
    timeout_seconds INTEGER NOT NULL DEFAULT 30,
    memory_mb INTEGER NOT NULL DEFAULT 128,
    max_concurrency INTEGER NOT NULL DEFAULT 10,
    environment JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT unique_function_name_version UNIQUE(name, version),
    CONSTRAINT check_timeout_positive CHECK (timeout_seconds > 0),
    CONSTRAINT check_memory_positive CHECK (memory_mb > 0),
    CONSTRAINT check_concurrency_positive CHECK (max_concurrency > 0)
);

-- Invocations table
CREATE TABLE invocations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    function_id UUID NOT NULL REFERENCES functions(id) ON DELETE CASCADE,
    payload JSONB,
    headers JSONB DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    result JSONB,
    error_type VARCHAR(100),
    error_message TEXT,
    error_stack TEXT,
    duration_ns BIGINT,
    cpu_time_ns BIGINT,
    memory_peak BIGINT,
    network_in BIGINT,
    network_out BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT check_status_valid CHECK (status IN ('pending', 'running', 'completed', 'failed', 'timeout'))
);

-- Indexes for performance
CREATE INDEX idx_functions_name ON functions(name);
CREATE INDEX idx_functions_runtime ON functions(runtime);
CREATE INDEX idx_functions_created_at ON functions(created_at DESC);

CREATE INDEX idx_invocations_function_id ON invocations(function_id);
CREATE INDEX idx_invocations_status ON invocations(status);
CREATE INDEX idx_invocations_created_at ON invocations(created_at DESC);
CREATE INDEX idx_invocations_function_status ON invocations(function_id, status);

-- Users table (for authentication)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    roles TEXT[] DEFAULT ARRAY['user'],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Function permissions table
CREATE TABLE function_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    function_id UUID NOT NULL REFERENCES functions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission VARCHAR(20) NOT NULL,
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT unique_function_user_permission UNIQUE(function_id, user_id, permission),
    CONSTRAINT check_permission_valid CHECK (permission IN ('read', 'write', 'execute', 'admin'))
);

CREATE INDEX idx_function_permissions_function_id ON function_permissions(function_id);
CREATE INDEX idx_function_permissions_user_id ON function_permissions(user_id);

-- Updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to functions table
CREATE TRIGGER update_functions_updated_at BEFORE UPDATE ON functions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Apply trigger to users table
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
