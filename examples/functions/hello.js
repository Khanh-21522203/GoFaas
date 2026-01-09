#!/usr/bin/env node

function main() {
    // Get payload from environment variable
    const payloadStr = process.env.FUNCTION_PAYLOAD || '{}';
    
    let payload;
    try {
        payload = JSON.parse(payloadStr);
    } catch (e) {
        payload = {};
    }
    
    const name = payload.name || 'World';
    
    const response = {
        message: `Hello, ${name}!`
    };
    
    console.log(JSON.stringify(response));
}

main();
