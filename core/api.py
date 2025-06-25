from flask import Flask, request, jsonify, render_template_string
from flask_cors import CORS
import os
import json
from agent import agent, logger
import tempfile
import time
import requests

app = Flask(__name__)
CORS(app)

# HTML template for the web interface
# This would be better not only as a separate file, but making it fully separate using React would be even better
# And thanks Claude
WEB_INTERFACE_TEMPLATE = """
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CI/CD Agent Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { background: #333; color: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .card { background: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .status-success { color: #28a745; }
        .status-failed { color: #dc3545; }
        .status-running { color: #007bff; }
        .status-pending { color: #ffc107; }
        .btn { padding: 10px 20px; margin: 5px; border: none; border-radius: 4px; cursor: pointer; }
        .btn-primary { background: #007bff; color: white; }
        .btn-success { background: #28a745; color: white; }
        .btn-danger { background: #dc3545; color: white; }
        .btn-warning { background: #ffc107; color: black; }
        textarea { width: 100%; height: 200px; margin: 10px 0; }
        .pipeline-item { border-left: 4px solid #007bff; padding-left: 15px; margin: 10px 0; }
        .run-item { border-left: 4px solid #28a745; padding-left: 15px; margin: 10px 0; }
        .step-item { margin: 5px 0; padding: 10px; background: #f8f9fa; border-radius: 4px; }
        .logs { background: #000; color: #0f0; padding: 10px; font-family: monospace; height: 300px; overflow-y: auto; }
        .status-never_run { color: #6c757d; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 CI/CD Agent Dashboard</h1>
            <p>Manage and monitor your CI/CD pipelines</p>
        </div>
        
        <div class="card">
            <h2>📋 Create Pipeline</h2>
            <textarea id="pipelineJson" placeholder="Paste your pipeline JSON here...">
{
    "name": "Simple Test Pipeline",
    "version": "1.0.0",
    "description": "A simple pipeline for testing the CI/CD agent",
    "variables": {
        "PROJECT_NAME": "my-app",
        "VERSION": "1.0.0",
        "BUILD_NUMBER": "001"
    },
    "steps": [
        {
            "name": "Environment Check",
            "description": "Check the environment",
            "command": "echo 'Environment: '$(uname -a) && echo 'Current directory: '$(pwd)",
            "timeout": 30
        },
        {
            "name": "Variable Test",
            "description": "Test variable substitution",
            "command": "echo 'Building ${PROJECT_NAME} version ${VERSION} build ${BUILD_NUMBER}'",
            "timeout": 30
        },
        {
            "name": "Simulated Build",
            "description": "Simulate a build process",
            "command": "echo 'Starting build...' && sleep 2 && echo 'Build completed successfully!'",
            "timeout": 60
        }
    ]
}
            </textarea>
            <br>
            <button class="btn btn-primary" onclick="createPipeline()">Create Pipeline</button>
            <button class="btn btn-success" onclick="createAndRunPipeline()">Create & Run</button>
        </div>
        
        <div class="card">
            <h2>📊 Pipelines</h2>
            <button class="btn btn-warning" onclick="loadPipelines()">Refresh Pipelines</button>
            <div id="pipelinesList"></div>
        </div>
        
        <div class="card">
            <h2>🏃 Pipeline Runs</h2>
            <button class="btn btn-warning" onclick="loadRuns()">Refresh Runs</button>
            <div id="runsList"></div>
        </div>
        
        <div class="card">
            <h2>📝 Logs</h2>
            <div id="logs" class="logs">Logs will appear here...</div>
            <button class="btn btn-warning" onclick="clearLogs()">Clear Logs</button>
        </div>
    </div>

    <script>
        let logBuffer = [];
        
        function log(message) {
            const timestamp = new Date().toISOString();
            logBuffer.push(`[${timestamp}] ${message}`);
            if (logBuffer.length > 100) logBuffer.shift();
            document.getElementById('logs').innerHTML = logBuffer.join('\\n');
            document.getElementById('logs').scrollTop = document.getElementById('logs').scrollHeight;
        }
        
        function clearLogs() {
            logBuffer = [];
            document.getElementById('logs').innerHTML = 'Logs cleared...';
        }
        
        async function createPipeline() {
            try {
                const pipelineJson = document.getElementById('pipelineJson').value;
                const pipeline = JSON.parse(pipelineJson);
                
                const response = await fetch('/api/pipelines', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(pipeline)
                });
                
                const result = await response.json();
                if (response.ok) {
                    log(`✅ Pipeline created: ${result.pipeline_id}`);
                    loadPipelines();
                } else {
                    log(`❌ Failed to create pipeline: ${result.error}`);
                }
            } catch (error) {
                log(`❌ Error: ${error.message}`);
            }
        }
        
        async function createAndRunPipeline() {
            try {
                const pipelineJson = document.getElementById('pipelineJson').value;
                const pipeline = JSON.parse(pipelineJson);
                
                const response = await fetch('/api/pipelines/run', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(pipeline)
                });
                
                const result = await response.json();
                if (response.ok) {
                    log(`🚀 Pipeline created and started: ${result.pipeline_id}`);
                    loadPipelines();
                } else {
                    log(`❌ Failed to create and run pipeline: ${result.error}`);
                }
            } catch (error) {
                log(`❌ Error: ${error.message}`);
            }
        }
        
        async function runPipeline(pipelineId) {
            try {
                const response = await fetch(`/api/pipelines/${pipelineId}/run`, {
                    method: 'POST'
                });
                
                const result = await response.json();
                if (response.ok) {
                    log(`🚀 Pipeline started: ${pipelineId}`);
                    loadPipelines();
                } else {
                    log(`❌ Failed to start pipeline: ${result.error}`);
                }
            } catch (error) {
                log(`❌ Error: ${error.message}`);
            }
        }
        
        async function cancelPipeline(pipelineId) {
            try {
                const response = await fetch(`/api/pipelines/${pipelineId}/cancel`, {
                    method: 'POST'
                });
                
                const result = await response.json();
                if (response.ok) {
                    log(`🛑 Pipeline cancelled: ${pipelineId}`);
                    loadPipelines();
                } else {
                    log(`❌ Failed to cancel pipeline: ${result.error}`);
                }
            } catch (error) {
                log(`❌ Error: ${error.message}`);
            }
        }
        
        async function loadPipelines() {
            try {
                const response = await fetch('/api/pipelines');
                const pipelines = await response.json();
                
                const container = document.getElementById('pipelinesList');
                if (pipelines.length === 0) {
                    container.innerHTML = '<p>No pipelines found.</p>';
                    return;
                }
                
                let html = '';
                pipelines.forEach(pipeline => {
                    const statusClass = `status-${pipeline.status}`;
                    const statusEmoji = getStatusEmoji(pipeline.status);
                    html += `
                        <div class="pipeline-item">
                            <h3>${statusEmoji} ${pipeline.name} <span class="${statusClass}">[${pipeline.status}]</span></h3>
                            <p><strong>ID:</strong> ${pipeline.id}</p>
                            <p><strong>Description:</strong> ${pipeline.description || 'No description'}</p>
                            <p><strong>Created:</strong> ${pipeline.created_at}</p>
                            <p><strong>Active Runs:</strong> ${pipeline.active_runs || 0}</p>
                            <p><strong>Total Runs:</strong> ${pipeline.total_runs || 0}</p>
                            ${pipeline.last_run_at ? `<p><strong>Last Run:</strong> ${pipeline.last_run_at}</p>` : ''}
                            <div>
                                <button class="btn btn-success" onclick="runPipeline('${pipeline.id}')">▶️ Run</button>
                                <button class="btn btn-warning" onclick="showPipelineDetails('${pipeline.id}')">📋 Details</button>
                                <button class="btn btn-warning" onclick="showPipelineRuns('${pipeline.id}')">🏃 Runs</button>
                                <button class="btn btn-danger" onclick="cancelPipeline('${pipeline.id}')">🛑 Cancel All</button>
                                <button class="btn btn-danger" onclick="deletePipeline('${pipeline.id}')" style="background: #dc3545;">🗑️ Delete</button>
                            </div>
                        </div>
                    `;
                });
                
                container.innerHTML = html;
            } catch (error) {
                log(`❌ Error loading pipelines: ${error.message}`);
            }
        }
        
        async function showPipelineDetails(pipelineId) {
            try {
                const response = await fetch(`/api/pipelines/${pipelineId}`);
                const pipeline = await response.json();
                
                log(`📋 Pipeline Details: ${pipeline.name}`);
                log(`Status: ${pipeline.status}`);
                log(`Steps:`);
                pipeline.steps.forEach(step => {
                    log(`  - ${step.name}: ${step.status}`);
                    if (step.output) log(`    Output: ${step.output}`);
                    if (step.error) log(`    Error: ${step.error}`);
                });
            } catch (error) {
                log(`❌ Error loading pipeline details: ${error.message}`);
            }
        }
        
        async function deletePipeline(pipelineId) {
            if (!confirm('Are you sure you want to delete this pipeline? This action cannot be undone.')) {
                return;
            }
            
            try {
                const response = await fetch(`/api/pipelines/${pipelineId}`, {
                    method: 'DELETE'
                });
                
                const result = await response.json();
                if (response.ok) {
                    log(`🗑️ Pipeline deleted: ${pipelineId}`);
                    loadPipelines();
                } else {
                    log(`❌ Failed to delete pipeline: ${result.error}`);
                }
            } catch (error) {
                log(`❌ Error deleting pipeline: ${error.message}`);
            }
        }
        
        function getStatusEmoji(status) {
            const emojis = {
                'pending': '⏳',
                'running': '🚀',
                'success': '✅',
                'failed': '❌',
                'cancelled': '🛑',
                'never_run': '💤'
            };
            return emojis[status] || '❓';
        }
        
        async function loadRuns() {
            try {
                const response = await fetch('/api/runs');
                const runs = await response.json();
                
                const container = document.getElementById('runsList');
                if (runs.length === 0) {
                    container.innerHTML = '<p>No runs found.</p>';
                    return;
                }
                
                let html = '';
                runs.forEach(run => {
                    const statusClass = `status-${run.status}`;
                    const statusEmoji = getStatusEmoji(run.status);
                    html += `
                        <div class="run-item">
                            <h4>${statusEmoji} ${run.name} <span class="${statusClass}">[${run.status}]</span></h4>
                            <p><strong>Run ID:</strong> ${run.id}</p>
                            <p><strong>Pipeline ID:</strong> ${run.pipeline_id}</p>
                            <p><strong>Created:</strong> ${run.created_at}</p>
                            ${run.started_at ? `<p><strong>Started:</strong> ${run.started_at}</p>` : ''}
                            ${run.finished_at ? `<p><strong>Finished:</strong> ${run.finished_at}</p>` : ''}
                            ${run.total_duration ? `<p><strong>Duration:</strong> ${run.total_duration.toFixed(2)}s</p>` : ''}
                            <div>
                                <button class="btn btn-warning" onclick="showRunDetails('${run.id}')">📋 Details</button>
                                ${run.status === 'running' ? `<button class="btn btn-danger" onclick="cancelRun('${run.id}')">🛑 Cancel</button>` : ''}
                                <button class="btn btn-danger" onclick="deleteRun('${run.id}')" style="background: #dc3545;">🗑️ Delete</button>
                            </div>
                        </div>
                    `;
                });
                
                container.innerHTML = html;
            } catch (error) {
                log(`❌ Error loading runs: ${error.message}`);
            }
        }
        
        async function showPipelineRuns(pipelineId) {
            try {
                const response = await fetch(`/api/runs?pipeline_id=${pipelineId}`);
                const runs = await response.json();
                
                log(`🏃 Runs for pipeline ${pipelineId}:`);
                if (runs.length === 0) {
                    log('No runs found for this pipeline.');
                    return;
                }
                
                runs.forEach(run => {
                    const statusEmoji = getStatusEmoji(run.status);
                    log(`${statusEmoji} ${run.name} [${run.status}] - Run ID: ${run.id}`);
                    log(`   Created: ${run.created_at}`);
                    if (run.started_at) log(`   Started: ${run.started_at}`);
                    if (run.finished_at) log(`   Finished: ${run.finished_at}`);
                    if (run.total_duration) log(`   Duration: ${run.total_duration.toFixed(2)}s`);
                });
            } catch (error) {
                log(`❌ Error loading pipeline runs: ${error.message}`);
            }
        }
        
        async function showRunDetails(runId) {
            try {
                const response = await fetch(`/api/runs/${runId}`);
                const run = await response.json();
                
                const statusEmoji = getStatusEmoji(run.status);
                log(`🏃 Run Details: ${run.name}`);
                log(`${statusEmoji} Status: ${run.status}`);
                log(`Run ID: ${run.id}`);
                log(`Pipeline ID: ${run.pipeline_id}`);
                log(`Created: ${run.created_at}`);
                if (run.started_at) log(`Started: ${run.started_at}`);
                if (run.finished_at) log(`Finished: ${run.finished_at}`);
                if (run.total_duration) log(`Duration: ${run.total_duration.toFixed(2)}s`);
                
                if (run.steps && run.steps.length > 0) {
                    log(`Steps:`);
                    run.steps.forEach(step => {
                        const stepEmoji = getStatusEmoji(step.status);
                        log(`  ${stepEmoji} ${step.name}: ${step.status}`);
                        if (step.output) log(`    Output: ${step.output}`);
                        if (step.error) log(`    Error: ${step.error}`);
                    });
                }
            } catch (error) {
                log(`❌ Error loading run details: ${error.message}`);
            }
        }
        
        async function cancelRun(runId) {
            try {
                const response = await fetch(`/api/runs/${runId}/cancel`, {
                    method: 'POST'
                });
                
                const result = await response.json();
                if (response.ok) {
                    log(`🛑 Run cancelled: ${runId}`);
                    loadRuns();
                    loadPipelines();
                } else {
                    log(`❌ Failed to cancel run: ${result.error}`);
                }
            } catch (error) {
                log(`❌ Error cancelling run: ${error.message}`);
            }
        }
        
        async function deleteRun(runId) {
            if (!confirm('Are you sure you want to delete this run? This action cannot be undone.')) {
                return;
            }
            
            try {
                const response = await fetch(`/api/runs/${runId}`, {
                    method: 'DELETE'
                });
                
                const result = await response.json();
                if (response.ok) {
                    log(`🗑️ Run deleted: ${runId}`);
                    loadRuns();
                    loadPipelines();
                } else {
                    log(`❌ Failed to delete run: ${result.error}`);
                }
            } catch (error) {
                log(`❌ Error deleting run: ${error.message}`);
            }
        }
        
        // Load data on page load
        loadPipelines();
        loadRuns();
        
        // Auto-refresh data every 5 seconds
        setInterval(() => {
            loadPipelines();
            loadRuns();
        }, 5000);
    </script>
</body>
</html>
"""

@app.route('/')
def dashboard():
    """Web dashboard for the CI/CD agent"""
    return render_template_string(WEB_INTERFACE_TEMPLATE)

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
