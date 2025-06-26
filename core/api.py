from flask import Flask, request, jsonify
from flask_cors import CORS
import os
import json
from agent import agent, logger
import tempfile
import time
import requests

app = Flask(__name__)
CORS(app)

@app.route('/api/pipelines', methods=['GET'])
def list_pipelines():
    """List all pipelines"""
    try:
        pipelines = agent.list_pipelines()
        return jsonify(pipelines)
    except Exception as e:
        logger.error(f"Error listing pipelines: {str(e)}")
        return jsonify({'error': str(e)}), requests.codes.internal_server_error

@app.route('/api/pipelines', methods=['POST'])
def create_pipeline():
    """Create a new pipeline"""
    try:
        pipeline_config = request.get_json()
        if not pipeline_config:
            return jsonify({'error': 'No pipeline configuration provided'}), requests.codes.bad_request
        
        pipeline_id = agent.create_pipeline(pipeline_config)
        return jsonify({'pipeline_id': pipeline_id, 'status': 'created'})
    except Exception as e:
        logger.error(f"Error creating pipeline: {str(e)}")
        return jsonify({'error': str(e)}), requests.codes.internal_server_error

@app.route('/api/pipelines/run', methods=['POST'])
def create_and_run_pipeline():
    """Create and immediately run a pipeline"""
    try:
        pipeline_config = request.get_json()
        if not pipeline_config:
            return jsonify({'error': 'No pipeline configuration provided'}), requests.codes.bad_request
        
        pipeline_id = agent.create_pipeline(pipeline_config)
        run_id = agent.run_pipeline(pipeline_id, background=True)
        
        if run_id:
            return jsonify({'pipeline_id': pipeline_id, 'run_id': run_id, 'status': 'running'})
        else:
            return jsonify({'error': 'Failed to start pipeline'}), requests.codes.internal_server_error
    except Exception as e:
        logger.error(f"Error creating and running pipeline: {str(e)}")
        return jsonify({'error': str(e)}), requests.codes.internal_server_error

@app.route('/api/pipelines/<pipeline_id>', methods=['GET'])
def get_pipeline_status(pipeline_id):
    """Get pipeline status and details"""
    try:
        status = agent.get_pipeline_status(pipeline_id)
        if status:
            return jsonify(status)
        else:
            return jsonify({'error': 'Pipeline not found'}), requests.codes.not_found
    except Exception as e:
        logger.error(f"Error getting pipeline status: {str(e)}")
        return jsonify({'error': str(e)}), requests.codes.internal_server_error

@app.route('/api/pipelines/<pipeline_id>/run', methods=['POST'])
def run_pipeline(pipeline_id):
    """Run an existing pipeline"""
    try:
        background = request.args.get('background', 'true').lower() == 'true'
        run_id = agent.run_pipeline(pipeline_id, background=background)
        
        if run_id:
            return jsonify({'run_id': run_id, 'status': 'running' if background else 'completed'})
        else:
            return jsonify({'error': 'Failed to start pipeline'}), requests.codes.internal_server_error
    except Exception as e:
        logger.error(f"Error running pipeline: {str(e)}")
        return jsonify({'error': str(e)}), requests.codes.internal_server_error

@app.route('/api/pipelines/<pipeline_id>/cancel', methods=['POST'])
def cancel_pipeline(pipeline_id):
    """Cancel a running pipeline"""
    try:
        success = agent.cancel_pipeline(pipeline_id)
        if success:
            return jsonify({'status': 'cancelled'})
        else:
            return jsonify({'error': 'Pipeline not found or not running'}), requests.codes.not_found
    except Exception as e:
        logger.error(f"Error cancelling pipeline: {str(e)}")
        return jsonify({'error': str(e)}), requests.codes.internal_server_error

@app.route('/api/pipelines/<pipeline_id>', methods=['DELETE'])
def delete_pipeline(pipeline_id):
    """Delete a pipeline"""
    try:
        success = agent.delete_pipeline(pipeline_id)
        if success:
            return jsonify({'status': 'deleted'})
        else:
            return jsonify({'error': 'Pipeline not found or cannot be deleted (may be running)'}), requests.codes.not_found
    except Exception as e:
        logger.error(f"Error deleting pipeline: {str(e)}")
        return jsonify({'error': str(e)}), requests.codes.internal_server_error

@app.route('/api/runs', methods=['GET'])
def list_runs():
    """List all runs, optionally filtered by pipeline_id"""
    try:
        pipeline_id = request.args.get('pipeline_id')
        runs = agent.list_runs(pipeline_id)
        return jsonify(runs)
    except Exception as e:
        logger.error(f"Error listing runs: {str(e)}")
        return jsonify({'error': str(e)}), requests.codes.internal_server_error

@app.route('/api/runs/<run_id>', methods=['GET'])
def get_run_status(run_id):
    """Get run status and details"""
    try:
        run_status = agent.get_run_status(run_id)
        if run_status:
            return jsonify(run_status)
        else:
            return jsonify({'error': 'Run not found'}), requests.codes.not_found
    except Exception as e:
        logger.error(f"Error getting run status for {run_id}: {str(e)}")
        return jsonify({'error': str(e)}), requests.codes.internal_server_error

@app.route('/api/runs/<run_id>/cancel', methods=['POST'])
def cancel_run(run_id):
    """Cancel a running pipeline run"""
    try:
        success = agent.cancel_run(run_id)
        if success:
            return jsonify({'status': 'cancelled'})
        else:
            return jsonify({'error': 'Run not found or not running'}), requests.codes.not_found
    except Exception as e:
        logger.error(f"Error cancelling run {run_id}: {str(e)}")
        return jsonify({'error': str(e)}), requests.codes.internal_server_error

@app.route('/api/runs/<run_id>', methods=['DELETE'])
def delete_run(run_id):
    """Delete a specific run"""
    try:
        success = agent.delete_run(run_id)
        if success:
            return jsonify({'status': 'deleted'})
        else:
            return jsonify({'error': 'Run not found or currently running'}), requests.codes.not_found
    except Exception as e:
        logger.error(f"Error deleting run {run_id}: {str(e)}")
        return jsonify({'error': str(e)}), requests.codes.internal_server_error

@app.route('/api/pipelines/upload', methods=['POST'])
def upload_pipeline():
    """Upload and create pipeline from file"""
    try:
        if 'file' not in request.files:
            return jsonify({'error': 'No file provided'}), requests.codes.bad_request
        
        file = request.files['file']
        if file.filename == '':
            return jsonify({'error': 'No file selected'}), requests.codes.bad_request
        
        with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as tmp_file:
            file_content = file.read().decode('utf-8')
            tmp_file.write(file_content)
            tmp_file_path = tmp_file.name
        
        pipeline_id = agent.load_pipeline_from_file(tmp_file_path)
        
        os.unlink(tmp_file_path)
        
        return jsonify({'pipeline_id': pipeline_id, 'status': 'created'})
    except Exception as e:
        logger.error(f"Error uploading pipeline: {str(e)}")
        return jsonify({'error': str(e)}), requests.codes.internal_server_error

@app.route('/health')
def health_check():
    """Health check endpoint"""
    return jsonify({
        'status': 'healthy',
        'timestamp': json.dumps(time.time()),
        'agent_status': 'running'
    })

if __name__ == '__main__':
    os.makedirs('/app/logs', exist_ok=True)
    
    port = int(os.environ.get('PORT', 8080))
    debug = os.environ.get('DEBUG', 'false').lower() == 'true'
    
    logger.info(f"Starting CI/CD Agent API on port {port}")
    app.run(host='0.0.0.0', port=port, debug=debug) 
