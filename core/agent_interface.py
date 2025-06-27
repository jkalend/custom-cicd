#!/usr/bin/env python3
"""
Agent Interface Script
Provides a command-line interface for the CI/CD agent that can be called from Next.js API routes.
"""

import sys
import json
import traceback
import logging

for handler in logging.root.handlers[:]:
    logging.root.removeHandler(handler)

logging.basicConfig(
    level=logging.ERROR,  
    stream=sys.stderr,    
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

logging.getLogger().handlers.clear()
logging.getLogger().setLevel(logging.ERROR)

stderr_handler = logging.StreamHandler(sys.stderr)
stderr_handler.setLevel(logging.ERROR)
logging.getLogger().addHandler(stderr_handler)

logger = logging.getLogger(__name__)

from agent import agent

def handle_command(command, args=None):
    """Handle agent commands and return JSON responses"""
    try:
        if command == 'list_pipelines':
            result = agent.list_pipelines()
            return {'success': True, 'data': result}
        
        elif command == 'create_pipeline':
            pipeline_config = args.get('pipeline_config') if args else None
            if not pipeline_config:
                return {'success': False, 'error': 'pipeline_config required'}
            pipeline_id = agent.create_pipeline(pipeline_config)
            return {'success': True, 'data': {'pipeline_id': pipeline_id, 'status': 'created'}}
        
        elif command == 'create_and_run_pipeline':
            pipeline_config = args.get('pipeline_config') if args else None
            if not pipeline_config:
                return {'success': False, 'error': 'pipeline_config required'}
            pipeline_id = agent.create_pipeline(pipeline_config)
            run_id = agent.run_pipeline(pipeline_id, background=True)
            return {'success': True, 'data': {'pipeline_id': pipeline_id, 'run_id': run_id, 'status': 'running'}}
        
        elif command == 'get_pipeline':
            pipeline_id = args.get('pipeline_id')
            if not pipeline_id:
                return {'success': False, 'error': 'pipeline_id required'}
            result = agent.get_pipeline_status(pipeline_id)
            if result:
                return {'success': True, 'data': result}
            else:
                return {'success': False, 'error': 'Pipeline not found'}
        
        elif command == 'run_pipeline':
            pipeline_id = args.get('pipeline_id')
            background = args.get('background', True)
            if not pipeline_id:
                return {'success': False, 'error': 'pipeline_id required'}
            run_id = agent.run_pipeline(pipeline_id, background=background)
            if run_id:
                return {'success': True, 'data': {'run_id': run_id, 'status': 'running' if background else 'completed'}}
            else:
                return {'success': False, 'error': 'Failed to start pipeline'}
        
        elif command == 'cancel_pipeline':
            pipeline_id = args.get('pipeline_id')
            if not pipeline_id:
                return {'success': False, 'error': 'pipeline_id required'}
            success = agent.cancel_pipeline(pipeline_id)
            if success:
                return {'success': True, 'data': {'status': 'cancelled'}}
            else:
                return {'success': False, 'error': 'Pipeline not found or not running'}
        
        elif command == 'delete_pipeline':
            pipeline_id = args.get('pipeline_id')
            if not pipeline_id:
                return {'success': False, 'error': 'pipeline_id required'}
            success = agent.delete_pipeline(pipeline_id)
            if success:
                return {'success': True, 'data': {'status': 'deleted'}}
            else:
                return {'success': False, 'error': 'Pipeline not found or cannot be deleted'}
        
        elif command == 'list_runs':
            pipeline_id = args.get('pipeline_id') if args else None
            result = agent.list_runs(pipeline_id)
            return {'success': True, 'data': result}
        
        elif command == 'get_run':
            run_id = args.get('run_id')
            if not run_id:
                return {'success': False, 'error': 'run_id required'}
            result = agent.get_run_status(run_id)
            if result:
                return {'success': True, 'data': result}
            else:
                return {'success': False, 'error': 'Run not found'}
        
        elif command == 'cancel_run':
            run_id = args.get('run_id')
            if not run_id:
                return {'success': False, 'error': 'run_id required'}
            success = agent.cancel_run(run_id)
            if success:
                return {'success': True, 'data': {'status': 'cancelled'}}
            else:
                return {'success': False, 'error': 'Run not found or not running'}
        
        elif command == 'delete_run':
            run_id = args.get('run_id')
            if not run_id:
                return {'success': False, 'error': 'run_id required'}
            success = agent.delete_run(run_id)
            if success:
                return {'success': True, 'data': {'status': 'deleted'}}
            else:
                return {'success': False, 'error': 'Run not found or currently running'}
        
        elif command == 'execute_run':
            run_id = args.get('run_id')
            if not run_id:
                return {'success': False, 'error': 'run_id required'}
            success = agent._execute_run(run_id)
            if success:
                return {'success': True, 'data': {'status': 'completed'}}
            else:
                return {'success': False, 'error': 'Execution failed'}
        
        elif command == 'health_check':
            return {
                'success': True, 
                'data': {
                    'status': 'healthy',
                    'timestamp': str(time.time()),
                    'agent_status': 'running'
                }
            }
        
        else:
            return {'success': False, 'error': f'Unknown command: {command}'}
    
    except Exception as e:
        logger.error(f"Error executing command {command}: {str(e)}")
        logger.error(traceback.format_exc())
        return {'success': False, 'error': str(e)}

def main():
    """Main entry point"""
    if len(sys.argv) < 2:
        return json.dumps({'success': False, 'error': 'Command required'})
    
    command = sys.argv[1]
    args = None
    
    if len(sys.argv) > 2:
        try:
            args = json.loads(sys.argv[2])
        except json.JSONDecodeError as e:
            return json.dumps({'success': False, 'error': f'Invalid JSON arguments: {str(e)}'})
    
    result = handle_command(command, args)
    print(json.dumps(result))

if __name__ == '__main__':
    import time
    main() 
