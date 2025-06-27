#!/usr/bin/env python3
"""
HTTP API wrapper for the agent interface.
Provides REST endpoints for the CI/CD pipeline agent.
"""
import json
import logging
import subprocess
from flask import Flask, request, jsonify
from flask_cors import CORS
import os
from http import HTTPStatus

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = Flask(__name__)
CORS(app)

def execute_agent_command(command, **kwargs):
    """Execute agent interface command and return JSON response."""
    try:
        cmd = ['python', 'agent_interface.py', command]
        
        if kwargs:
            cmd.append(json.dumps(kwargs))
        
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=30,
            cwd=os.path.dirname(os.path.abspath(__file__))
        )
        
        if result.returncode != 0:
            logger.error(f"Command failed: {result.stderr}")
            return {'error': result.stderr}, HTTPStatus.INTERNAL_SERVER_ERROR
        
        # Parse JSON response - extract only JSON part from output (if agent by any change produce additional stdout)
        try:
            stdout = result.stdout.strip()
            
            # Find the JSON object by looking for balanced braces
            json_start = stdout.find('{')
            if json_start != -1:
                brace_count = 0
                json_end = json_start
                
                for i in range(json_start, len(stdout)):
                    if stdout[i] == '{':
                        brace_count += 1
                    elif stdout[i] == '}':
                        brace_count -= 1
                        if brace_count == 0:
                            json_end = i + 1
                            break
                
                if brace_count == 0:
                    json_part = stdout[json_start:json_end]
                    return json.loads(json_part), HTTPStatus.OK
            
            # If extraction fails, try to parse the entire stdout
            return json.loads(stdout), HTTPStatus.OK
                
        except json.JSONDecodeError as e:
            logger.error(f"Failed to parse JSON response: {e}")
            logger.error(f"Raw stdout (first 200 chars): {repr(result.stdout[:200])}")
            return {'error': 'Invalid JSON response from agent'}, HTTPStatus.INTERNAL_SERVER_ERROR
            
    except subprocess.TimeoutExpired:
        logger.error("Command timed out")
        return {'error': 'Command timed out'}, HTTPStatus.INTERNAL_SERVER_ERROR
    except Exception as e:
        logger.error(f"Unexpected error: {e}")
        return {'error': str(e)}, HTTPStatus.INTERNAL_SERVER_ERROR

@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint."""
    response, status = execute_agent_command('health_check')
    return jsonify(response), status

@app.route('/pipelines', methods=['GET'])
def list_pipelines():
    """List all pipelines."""
    response, status = execute_agent_command('list_pipelines')
    return jsonify(response), status

@app.route('/pipelines', methods=['POST'])
def create_pipeline():
    """Create a new pipeline."""
    data = request.get_json()
    if not data:
        return jsonify({'error': 'No JSON data provided'}), HTTPStatus.BAD_REQUEST
    
    response, status = execute_agent_command('create_pipeline', pipeline_config=data)
    return jsonify(response), status

@app.route('/pipelines/<pipeline_id>', methods=['GET'])
def get_pipeline(pipeline_id):
    """Get a specific pipeline."""
    response, status = execute_agent_command('get_pipeline', pipeline_id=pipeline_id)
    return jsonify(response), status

@app.route('/pipelines/<pipeline_id>', methods=['DELETE'])
def delete_pipeline(pipeline_id):
    """Delete a specific pipeline."""
    response, status = execute_agent_command('delete_pipeline', pipeline_id=pipeline_id)
    return jsonify(response), status

@app.route('/pipelines/<pipeline_id>/run', methods=['POST'])
def run_pipeline(pipeline_id):
    """Run a specific pipeline."""
    response, status = execute_agent_command('run_pipeline', pipeline_id=pipeline_id)
    return jsonify(response), status

@app.route('/pipelines/<pipeline_id>/cancel', methods=['POST'])
def cancel_pipeline(pipeline_id):
    """Cancel a running pipeline."""
    response, status = execute_agent_command('cancel_pipeline', pipeline_id=pipeline_id)
    return jsonify(response), status

@app.route('/pipelines/run', methods=['POST'])
def create_and_run_pipeline():
    """Create and run a pipeline in one step."""
    data = request.get_json()
    if not data:
        return jsonify({'error': 'No JSON data provided'}), HTTPStatus.BAD_REQUEST
    
    create_response, create_status = execute_agent_command('create_pipeline', pipeline_config=data)
    if create_status != HTTPStatus.OK:
        return jsonify(create_response), create_status
    
    pipeline_id = create_response.get('data', {}).get('pipeline_id')
    if not pipeline_id:
        return jsonify({'error': 'Failed to get pipeline ID from creation response'}), HTTPStatus.INTERNAL_SERVER_ERROR
    
    run_response, run_status = execute_agent_command('run_pipeline', pipeline_id=pipeline_id)
    return jsonify(run_response), run_status

@app.route('/runs', methods=['GET'])
def list_runs():
    """List all runs, optionally filtered by pipeline_id."""
    pipeline_id = request.args.get('pipeline_id')
    kwargs = {'pipeline_id': pipeline_id} if pipeline_id else {}
    response, status = execute_agent_command('list_runs', **kwargs)
    return jsonify(response), status

@app.route('/runs/<run_id>', methods=['GET'])
def get_run(run_id):
    """Get a specific run."""
    response, status = execute_agent_command('get_run', run_id=run_id)
    return jsonify(response), status

@app.route('/runs/<run_id>', methods=['DELETE'])
def delete_run(run_id):
    """Delete a specific run."""
    response, status = execute_agent_command('delete_run', run_id=run_id)
    return jsonify(response), status

@app.route('/runs/<run_id>/cancel', methods=['POST'])
def cancel_run(run_id):
    """Cancel a specific run."""
    response, status = execute_agent_command('cancel_run', run_id=run_id)
    return jsonify(response), status

@app.errorhandler(HTTPStatus.NOT_FOUND)
def not_found(error):
    """Handle 404 errors."""
    return jsonify({'error': 'Endpoint not found'}), HTTPStatus.NOT_FOUND

@app.errorhandler(HTTPStatus.INTERNAL_SERVER_ERROR)
def internal_error(error):
    """Handle 500 errors."""
    logger.error(f"Internal server error: {error}")
    return jsonify({'error': 'Internal server error'}), HTTPStatus.INTERNAL_SERVER_ERROR

if __name__ == '__main__':
    port = int(os.environ.get('PORT', 8000))
    debug = os.environ.get('DEBUG', 'false').lower() == 'true'
    
    logger.info(f"Starting backend API on port {port}")
    app.run(host='0.0.0.0', port=port, debug=debug) 
