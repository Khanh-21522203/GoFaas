#!/usr/bin/env python3
import json
import os

def main():
    # Get payload from environment variable
    payload_str = os.environ.get('FUNCTION_PAYLOAD', '{}')
    
    try:
        payload = json.loads(payload_str)
    except:
        payload = {}
    
    name = payload.get('name', 'World')
    
    response = {
        'message': f'Hello, {name}!'
    }
    
    print(json.dumps(response))

if __name__ == '__main__':
    main()
