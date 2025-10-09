"""
Flask CRUD Todo API with comprehensive logging
Stores todos in-memory and logs all operations
"""
import logging
import logging.handlers
import json
import os
from datetime import datetime
from typing import Dict, List, Optional
from flask import Flask, request, jsonify
from werkzeug.exceptions import HTTPException
import sys

# Configure comprehensive logging
def setup_logging():
    """Setup scalable logging with rotation and structured output"""
    log_level = os.getenv('LOG_LEVEL', 'INFO').upper()

    # Create logger
    logger = logging.getLogger('todo_api')
    logger.setLevel(getattr(logging, log_level))

    # Remove existing handlers to avoid duplicates
    logger.handlers = []

    # Console handler with JSON formatting for structured logs
    console_handler = logging.StreamHandler(sys.stdout)
    console_handler.setLevel(getattr(logging, log_level))

    # Custom JSON formatter for structured logging
    class JSONFormatter(logging.Formatter):
        def format(self, record):
            log_data = {
                'timestamp': datetime.utcnow().isoformat() + 'Z',
                'level': record.levelname,
                'logger': record.name,
                'message': record.getMessage(),
                'module': record.module,
                'function': record.funcName,
                'line': record.lineno,
            }

            # Add extra fields if present
            if hasattr(record, 'extra_data'):
                log_data['data'] = record.extra_data

            if record.exc_info:
                log_data['exception'] = self.formatException(record.exc_info)

            return json.dumps(log_data)

    console_handler.setFormatter(JSONFormatter())
    logger.addHandler(console_handler)

    return logger

# Initialize logger
logger = setup_logging()

# Initialize Flask app
app = Flask(__name__)

# In-memory data store
todos: Dict[int, Dict] = {}
next_id: int = 1

def log_operation(operation: str, data: Optional[Dict] = None, result: Optional[Dict] = None):
    """Helper function to log operations with payload"""
    extra_data = {
        'operation': operation,
        'timestamp': datetime.utcnow().isoformat() + 'Z',
    }

    if data:
        extra_data['input_payload'] = data

    if result:
        extra_data['result'] = result

    logger.info(f"Operation: {operation}", extra={'extra_data': extra_data})

@app.before_request
def log_request():
    """Log incoming request details"""
    request_data = {
        'method': request.method,
        'path': request.path,
        'remote_addr': request.remote_addr,
        'user_agent': request.headers.get('User-Agent', 'Unknown'),
    }

    if request.is_json:
        request_data['body'] = request.get_json(silent=True)
    elif request.data:
        request_data['body'] = request.data.decode('utf-8', errors='ignore')

    log_operation('incoming_request', data=request_data)

@app.after_request
def log_response(response):
    """Log outgoing response details"""
    response_data = {
        'status_code': response.status_code,
        'content_type': response.content_type,
    }

    log_operation('outgoing_response', data=response_data)
    return response

@app.errorhandler(Exception)
def handle_error(error):
    """Global error handler with logging"""
    error_data = {
        'error_type': type(error).__name__,
        'error_message': str(error),
    }

    if isinstance(error, HTTPException):
        error_data['status_code'] = error.code
        log_operation('http_error', data=error_data)
        return jsonify({'error': error.description}), error.code

    logger.error(f"Unhandled exception: {error}", extra={'extra_data': error_data}, exc_info=True)
    return jsonify({'error': 'Internal server error'}), 500

@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint"""
    health_data = {
        'status': 'healthy',
        'timestamp': datetime.utcnow().isoformat() + 'Z',
        'todos_count': len(todos),
    }
    log_operation('health_check', result=health_data)
    return jsonify(health_data), 200

@app.route('/todos', methods=['GET'])
def get_todos():
    """Get all todos"""
    todos_list = list(todos.values())
    log_operation('get_all_todos', result={'count': len(todos_list), 'todos': todos_list})
    return jsonify({'todos': todos_list, 'count': len(todos_list)}), 200

@app.route('/todos/<int:todo_id>', methods=['GET'])
def get_todo(todo_id: int):
    """Get a specific todo by ID"""
    if todo_id not in todos:
        log_operation('get_todo_not_found', data={'todo_id': todo_id})
        return jsonify({'error': f'Todo with id {todo_id} not found'}), 404

    todo = todos[todo_id]
    log_operation('get_todo', data={'todo_id': todo_id}, result=todo)
    return jsonify(todo), 200

@app.route('/todos', methods=['POST'])
def create_todo():
    """Create a new todo"""
    global next_id

    if not request.is_json:
        log_operation('create_todo_invalid_content_type', data={'content_type': request.content_type})
        return jsonify({'error': 'Content-Type must be application/json'}), 400

    data = request.get_json()

    # Validate required fields
    if not data or 'title' not in data:
        log_operation('create_todo_missing_title', data=data)
        return jsonify({'error': 'Missing required field: title'}), 400

    # Create new todo
    todo = {
        'id': next_id,
        'title': data['title'],
        'description': data.get('description', ''),
        'completed': data.get('completed', False),
        'created_at': datetime.utcnow().isoformat() + 'Z',
        'updated_at': datetime.utcnow().isoformat() + 'Z',
    }

    todos[next_id] = todo
    log_operation('create_todo', data=data, result=todo)

    next_id += 1

    return jsonify(todo), 201

@app.route('/todos/<int:todo_id>', methods=['PUT'])
def update_todo(todo_id: int):
    """Update an existing todo"""
    if todo_id not in todos:
        log_operation('update_todo_not_found', data={'todo_id': todo_id})
        return jsonify({'error': f'Todo with id {todo_id} not found'}), 404

    if not request.is_json:
        log_operation('update_todo_invalid_content_type', data={'content_type': request.content_type})
        return jsonify({'error': 'Content-Type must be application/json'}), 400

    data = request.get_json()
    old_todo = todos[todo_id].copy()

    # Update fields
    if 'title' in data:
        todos[todo_id]['title'] = data['title']
    if 'description' in data:
        todos[todo_id]['description'] = data['description']
    if 'completed' in data:
        todos[todo_id]['completed'] = data['completed']

    todos[todo_id]['updated_at'] = datetime.utcnow().isoformat() + 'Z'

    log_operation('update_todo', data={'todo_id': todo_id, 'old': old_todo, 'new': data}, result=todos[todo_id])

    return jsonify(todos[todo_id]), 200

@app.route('/todos/<int:todo_id>', methods=['PATCH'])
def patch_todo(todo_id: int):
    """Partially update an existing todo"""
    if todo_id not in todos:
        log_operation('patch_todo_not_found', data={'todo_id': todo_id})
        return jsonify({'error': f'Todo with id {todo_id} not found'}), 404

    if not request.is_json:
        log_operation('patch_todo_invalid_content_type', data={'content_type': request.content_type})
        return jsonify({'error': 'Content-Type must be application/json'}), 400

    data = request.get_json()
    old_todo = todos[todo_id].copy()

    # Update only provided fields
    if 'title' in data:
        todos[todo_id]['title'] = data['title']
    if 'description' in data:
        todos[todo_id]['description'] = data['description']
    if 'completed' in data:
        todos[todo_id]['completed'] = data['completed']

    todos[todo_id]['updated_at'] = datetime.utcnow().isoformat() + 'Z'

    log_operation('patch_todo', data={'todo_id': todo_id, 'old': old_todo, 'changes': data}, result=todos[todo_id])

    return jsonify(todos[todo_id]), 200

@app.route('/todos/<int:todo_id>', methods=['DELETE'])
def delete_todo(todo_id: int):
    """Delete a todo"""
    if todo_id not in todos:
        log_operation('delete_todo_not_found', data={'todo_id': todo_id})
        return jsonify({'error': f'Todo with id {todo_id} not found'}), 404

    deleted_todo = todos.pop(todo_id)
    log_operation('delete_todo', data={'todo_id': todo_id}, result=deleted_todo)

    return jsonify({'message': 'Todo deleted successfully', 'todo': deleted_todo}), 200

@app.route('/', methods=['GET'])
def root():
    """Root endpoint with API information"""
    api_info = {
        'name': 'Todo CRUD API',
        'version': '1.0.0',
        'endpoints': {
            'health': 'GET /health',
            'list_todos': 'GET /todos',
            'get_todo': 'GET /todos/:id',
            'create_todo': 'POST /todos',
            'update_todo': 'PUT /todos/:id',
            'patch_todo': 'PATCH /todos/:id',
            'delete_todo': 'DELETE /todos/:id',
        },
        'timestamp': datetime.utcnow().isoformat() + 'Z',
    }
    log_operation('api_info_requested', result=api_info)
    return jsonify(api_info), 200

if __name__ == '__main__':
    port = int(os.getenv('PORT', 5773))
    logger.info(f"Starting Todo API server on port {port}")
    app.run(host='0.0.0.0', port=port, debug=False)
