#!/usr/bin/env python3

import argparse
import json
import time
from agent import agent, PipelineStatus
import requests
import sys

class CICDCli:
    def __init__(self, api_url=None):
        # Auto-detect if we should use API mode when running in container
        if api_url is None:
            try:
                import requests
                response = requests.get("http://localhost:8080/health", timeout=2)
                if response.status_code == 200:
                    api_url = "http://localhost:8080"
                    print("🔗 Auto-detected API mode (API is available)")
            except Exception as e:
                print(f"❌ Error auto-detecting API mode: {str(e)}")
                print("🔗 Using direct mode")
        
        self.api_url = api_url or "http://localhost:8080"
        self.use_api = api_url is not None
        self.status_emoji = { # thank you copilot for the emojis
                'pending': '⏳',
                'running': '🚀',
                'success': '✅',
                'failed': '❌',
                'cancelled': '🛑',
                'skipped': '⏭️',
                'never_run': '💤'
            }
        
    def create_pipeline(self, pipeline_file: str) -> str | None:
        """Create a pipeline from a JSON file or stdin

        Args:
            pipeline_file (str): The path to the pipeline JSON file, or "-" for stdin
        Returns:
            The ID of the created pipeline
        """
        try:
            if self.use_api:
                return self._api_create_pipeline(pipeline_file)
            else:
                if pipeline_file == "-":
                    print("📥 Reading pipeline configuration from stdin...")
                    pipeline_config = json.load(sys.stdin)
                    pipeline_id = agent.create_pipeline(pipeline_config)
                else:
                    pipeline_id = agent.load_pipeline_from_file(pipeline_file)
                
                print("✅ Pipeline created successfully!")
                print(f"📋 Pipeline ID: {pipeline_id}")
                return pipeline_id
        except Exception as e:
            print(f"❌ Error creating pipeline: {str(e)}")
            return None
    
    def _api_create_pipeline(self, pipeline_file: str) -> str | None:
        """Create pipeline via API

        Args:
            pipeline_file (str): The path to the pipeline JSON file, or "-" for stdin
        Returns:
            The pipeline ID if created successfully, None otherwise
        """
        if pipeline_file == "-":
            print("📥 Reading pipeline configuration from stdin...")
            pipeline_config = json.load(sys.stdin)
        else:
            with open(pipeline_file, 'r') as f:
                pipeline_config = json.load(f)
        
        response = requests.post(f"{self.api_url}/api/pipelines", json=pipeline_config)
        if response.status_code == requests.codes.ok:
            result = response.json()
            print("✅ Pipeline created successfully!")
            print(f"📋 Pipeline ID: {result['pipeline_id']}")
            return result['pipeline_id']
        else:
            error = response.json().get('error', 'Unknown error')
            print(f"❌ Error creating pipeline: {error}")
            return None
    
    def run_pipeline(self, pipeline_id: str, background: bool = True) -> bool:
        """Run a pipeline by ID

        Args:
            pipeline_id (str): The ID of the pipeline to run
            background (bool): Whether to run the pipeline in background
        Returns:
            True if the pipeline was started successfully, False otherwise
        """
        try:
            if self.use_api:
                return self._api_run_pipeline(pipeline_id, background)
            else:
                print(f"🚀 Starting pipeline {pipeline_id}...")
                success = agent.run_pipeline(pipeline_id, background=background)
                if success:
                    if background:
                        print("✅ Pipeline started in background")
                    else:
                        print("✅ Pipeline completed successfully!")
                else:
                    print("❌ Pipeline failed")
                return success == PipelineStatus.SUCCESS
        except Exception as e:
            print(f"❌ Error running pipeline: {str(e)}")
            return False
    
    def _api_run_pipeline(self, pipeline_id: str, background: bool) -> bool:
        """Run pipeline via API

        Args:
            pipeline_id (str): The ID of the pipeline to run
            background (bool): Whether to run the pipeline in background
        Returns:
            True if the pipeline was started successfully, False otherwise
        """
        response = requests.post(f"{self.api_url}/api/pipelines/{pipeline_id}/run", 
                               params={'background': str(background).lower()})
        if response.status_code == requests.codes.ok:
            print("🚀 Pipeline started successfully!")
            return True
        else:
            error = response.json().get('error', 'Unknown error')
            print(f"❌ Error running pipeline: {error}")
            return False
    
    def create_and_run(self, pipeline_file: str, background: bool = True) -> bool:
        """Create and run a pipeline in one step

        Args:
            pipeline_file (str): The path to the pipeline JSON file, or "-" for stdin
            background (bool): Whether to run the pipeline in background
        Returns:
            True if the pipeline was created and started successfully, False otherwise
        """
        try:
            if self.use_api:
                return self._api_create_and_run(pipeline_file)
            else:
                pipeline_id = self.create_pipeline(pipeline_file)
                if pipeline_id:
                    return self.run_pipeline(pipeline_id, background)
                return False
        except Exception as e:
            print(f"❌ Error creating and running pipeline: {str(e)}")
            return False
    
    def _api_create_and_run(self, pipeline_file: str) -> bool:
        """Create and run pipeline via API

        Args:
            pipeline_file (str): The path to the pipeline JSON file, or "-" for stdin
        Returns:
            True if the pipeline was created and started successfully, False otherwise
        """
        if pipeline_file == "-":
            print("📥 Reading pipeline configuration from stdin...")
            pipeline_config = json.load(sys.stdin)
        else:
            with open(pipeline_file, 'r') as f:
                pipeline_config = json.load(f)
        
        response = requests.post(f"{self.api_url}/api/pipelines/run", json=pipeline_config)
        if response.status_code == requests.codes.ok:
            result = response.json()
            print("✅ Pipeline created and started successfully!")
            print(f"📋 Pipeline ID: {result['pipeline_id']}")
            return True
        else:
            error = response.json().get('error', 'Unknown error')
            print(f"❌ Error creating and running pipeline: {error}")
            return False
    
    def list_pipelines(self) -> list[dict]:
        """List all pipelines

        Returns:
            List of pipeline dictionaries
        """
        try:
            if self.use_api:
                return self._api_list_pipelines()
            else:
                pipelines = agent.list_pipelines()
                self._print_pipelines(pipelines)
                return pipelines
        except Exception as e:
            print(f"❌ Error listing pipelines: {str(e)}")
            return []
    
    def _api_list_pipelines(self) -> list[dict]:
        """List pipelines via API

        Returns:
            List of pipeline dictionaries
        """
        response = requests.get(f"{self.api_url}/api/pipelines")
        if response.status_code == requests.codes.ok:
            pipelines = response.json()
            self._print_pipelines(pipelines)
            return pipelines
        else:
            print("❌ Error listing pipelines")
            return []
    
    def _print_pipelines(self, pipelines: list[dict]):
        """Print pipelines in a formatted way"""
        if not pipelines:
            print("📋 No pipelines found")
            return
        
        print(f"\n📋 Found {len(pipelines)} pipeline(s):")
        print("-" * 80)
        for pipeline in pipelines:
            status_emoji = self.status_emoji.get(pipeline['status'], '❓')
            
            print(f"{status_emoji} {pipeline['name']}")
            print(f"\tID: {pipeline['id']}")
            print(f"\tStatus: {pipeline['status']}")
            print(f"\tCreated: {pipeline['created_at']}")
            if pipeline.get('started_at'):
                print(f"\tStarted: {pipeline['started_at']}")
            if pipeline.get('finished_at'):
                print(f"\tFinished: {pipeline['finished_at']}")
            print()
    
    def get_pipeline_status(self, pipeline_id: str) -> dict | None:
        """Get detailed pipeline status"""
        try:
            if self.use_api:
                return self._api_get_pipeline_status(pipeline_id)
            else:
                status = agent.get_pipeline_status(pipeline_id)
                if status:
                    self._print_pipeline_details(status)
                else:
                    print(f"❌ Pipeline {pipeline_id} not found")
                return status
        except Exception as e:
            print(f"❌ Error getting pipeline status: {str(e)}")
            return None
    
    def _api_get_pipeline_status(self, pipeline_id: str) -> dict | None:
        """Get pipeline status via API"""
        response = requests.get(f"{self.api_url}/api/pipelines/{pipeline_id}")
        if response.status_code == requests.codes.ok:
            status = response.json()
            self._print_pipeline_details(status)
            return status
        else:
            print(f"❌ Pipeline {pipeline_id} not found")
            return None
    
    def _print_pipeline_details(self, status: dict) -> None:
        """Print detailed pipeline information"""
        status_emoji = self.status_emoji.get(status['status'], '❓')
        
        print(f"\n{status_emoji} Pipeline: {status['name']}")
        print(f"📋 ID: {status['id']}")
        print(f"📊 Status: {status['status']}")
        if status.get('started_at'):
            print(f"🕐 Started: {status['started_at']}")
        if status.get('finished_at'):
            print(f"🕐 Finished: {status['finished_at']}")
        if status.get('total_duration'):
            print(f"⏱️  Duration: {status['total_duration']:.2f} seconds")
        
        print("\n📝 Steps:")
        for i, step in enumerate(status['steps'], 1):
            step_emoji = self.status_emoji.get(step['status'], '❓')
            
            print(f"  {i}. {step_emoji} {step['name']} [{step['status']}]")
            if step.get('output') and step['output'].strip():
                print(f"     📤 Output: {step['output'].strip()}")
            if step.get('error') and step['error'].strip():
                print(f"     🚨 Error: {step['error'].strip()}")
    
    def cancel_pipeline(self, pipeline_id: str) -> bool:
        """Cancel a running pipeline
        
        Args:
            pipeline_id (str): The ID of the pipeline to cancel
        Returns:
            bool: True if the pipeline was cancelled successfully, False otherwise
        """
        try:
            if self.use_api:
                return self._api_cancel_pipeline(pipeline_id)
            else:
                success = agent.cancel_pipeline(pipeline_id)
                if success:
                    print(f"🛑 Pipeline {pipeline_id} cancelled")
                else:
                    print(f"❌ Pipeline {pipeline_id} not found or not running")
                return success
        except Exception as e:
            print(f"❌ Error cancelling pipeline: {str(e)}")
            return False
    
    def _api_cancel_pipeline(self, pipeline_id: str) -> bool:
        """Cancel pipeline via API
        
        Args:
            pipeline_id (str): The ID of the pipeline to cancel
        Returns:
            bool: True if the pipeline was cancelled successfully, False otherwise
        """
        response = requests.post(f"{self.api_url}/api/pipelines/{pipeline_id}/cancel")
        if response.status_code == requests.codes.ok:
            print(f"🛑 Pipeline {pipeline_id} cancelled")
            return True
        else:
            print(f"❌ Pipeline {pipeline_id} not found or not running")
            return False
    
    def monitor_pipeline(self, pipeline_id: str, interval: int = 2) -> None:
        """Monitor a pipeline in real-time
        
        Args:
            pipeline_id (str): The ID of the pipeline to monitor
            interval (int): Refresh interval in seconds (default: 2)
        """
        print(f"👀 Monitoring pipeline {pipeline_id} (Ctrl+C to stop)")
        try:
            while True:
                status = self.get_pipeline_status(pipeline_id)
                if not status:
                    break
                
                if status['status'] in ['success', 'failed', 'cancelled']:
                    print(f"\n🏁 Pipeline finished with status: {status['status']}")
                    break
                
                time.sleep(interval)
                print("\n🔄 Refreshing... (Press Ctrl+C to stop monitoring)")
        
        except KeyboardInterrupt:
            print(f"\n👋 Stopped monitoring pipeline {pipeline_id}")

    def delete_pipeline(self, pipeline_id: str) -> bool:
        """Delete a pipeline
        
        Args:
            pipeline_id (str): The ID of the pipeline to delete
        Returns:
            bool: True if the pipeline was deleted successfully, False otherwise
        """
        try:
            if self.use_api:
                return self._api_delete_pipeline(pipeline_id)
            else:
                success = agent.delete_pipeline(pipeline_id)
                if success:
                    print(f"🗑️ Pipeline {pipeline_id} deleted")
                else:
                    print(f"❌ Pipeline {pipeline_id} not found or cannot be deleted (may be running)")
                return success
        except Exception as e:
            print(f"❌ Error deleting pipeline: {str(e)}")
            return False
    
    def _api_delete_pipeline(self, pipeline_id: str) -> bool:
        """Delete pipeline via API
        
        Args:
            pipeline_id (str): The ID of the pipeline to delete
        Returns:
            bool: True if the pipeline was deleted successfully, False otherwise
        """
        response = requests.delete(f"{self.api_url}/api/pipelines/{pipeline_id}")
        if response.status_code == requests.codes.ok:
            print(f"🗑️ Pipeline {pipeline_id} deleted")
            return True
        else:
            error = response.json().get('error', 'Unknown error')
            print(f"❌ Pipeline {pipeline_id} not found or cannot be deleted: {error}")
            return False
    
    def list_runs(self, pipeline_id: str | None = None) -> list[dict]:
        """List all runs, optionally filtered by pipeline ID
        
        Args:
            pipeline_id (str): Optional filter by pipeline ID
        Returns:
            list: List of run dictionaries
        """
        try:
            if self.use_api:
                return self._api_list_runs(pipeline_id)
            else:
                runs = agent.list_runs(pipeline_id)
                self._print_runs(runs)
                return runs
        except Exception as e:
            print(f"❌ Error listing runs: {str(e)}")
            return []
    
    def _api_list_runs(self, pipeline_id: str | None = None) -> list[dict]:
        """List runs via API
        
        Args:
            pipeline_id (str): Optional filter by pipeline ID
        Returns:
            list: List of run dictionaries
        """
        params = {'pipeline_id': pipeline_id} if pipeline_id else {}
        response = requests.get(f"{self.api_url}/api/runs", params=params)
        if response.status_code == requests.codes.ok:
            runs = response.json()
            self._print_runs(runs)
            return runs
        else:
            print("❌ Error listing runs")
            return []
    
    def _print_runs(self, runs: list[dict]) -> None:
        """Print runs in a formatted way"""
        if not runs:
            print("🏃 No runs found")
            return
        
        print(f"\n🏃 Found {len(runs)} run(s):")
        print("-" * 80)
        for run in runs:
            status_emoji = {
                'pending': '⏳',
                'running': '🚀',
                'success': '✅',
                'failed': '❌',
                'cancelled': '🛑'
            }.get(run['status'], '❓')
            
            print(f"{status_emoji} {run['name']}")
            print(f"   Run ID: {run['id']}")
            print(f"   Pipeline ID: {run['pipeline_id']}")
            print(f"   Status: {run['status']}")
            print(f"   Created: {run['created_at']}")
            if run.get('started_at'):
                print(f"   Started: {run['started_at']}")
            if run.get('finished_at'):
                print(f"   Finished: {run['finished_at']}")
            if run.get('total_duration'):
                print(f"   Duration: {run['total_duration']:.2f}s")
            print()
    
    def get_run_status(self, run_id: str) -> dict | None:
        """Get detailed run status
        
        Args:
            run_id (str): The ID of the run to get status for
        Returns:
            dict: Run status information
        """
        try:
            if self.use_api:
                return self._api_get_run_status(run_id)
            else:
                status = agent.get_run_status(run_id)
                if status:
                    self._print_run_details(status)
                else:
                    print(f"❌ Run {run_id} not found")
                return status
        except Exception as e:
            print(f"❌ Error getting run status: {str(e)}")
            return None
    
    def _api_get_run_status(self, run_id: str) -> dict | None:
        """Get run status via API
        
        Args:
            run_id (str): The ID of the run to get status for
        Returns:
            dict: Run status information
        """
        response = requests.get(f"{self.api_url}/api/runs/{run_id}")
        if response.status_code == requests.codes.ok:
            status = response.json()
            self._print_run_details(status)
            return status
        else:
            print(f"❌ Run {run_id} not found")
            return None
    
    def _print_run_details(self, status: dict) -> None:
        """Print detailed run information"""
        status_emoji = self.status_emoji.get(status['status'], '❓')
        
        print(f"\n{status_emoji} Run: {status['name']}")
        print(f"🏃 Run ID: {status['id']}")
        print(f"📋 Pipeline ID: {status['pipeline_id']}")
        print(f"📊 Status: {status['status']}")
        print(f"🕐 Created: {status['created_at']}")
        if status.get('started_at'):
            print(f"🕐 Started: {status['started_at']}")
        if status.get('finished_at'):
            print(f"🕐 Finished: {status['finished_at']}")
        if status.get('total_duration'):
            print(f"⏱️  Duration: {status['total_duration']:.3f}s")
        
        if status.get('steps'):
            print("\n📝 Steps:")
            for i, step in enumerate(status['steps'], 1):
                step_emoji = self.status_emoji.get(step['status'], '❓')
                
                print(f"  {i}. {step_emoji} {step['name']} [{step['status']}]")
                if step.get('output') and step['output'].strip():
                    print(f"     📤 Output: {step['output'].strip()}")
                if step.get('error') and step['error'].strip():
                    print(f"     🚨 Error: {step['error'].strip()}")
    
    def cancel_run(self, run_id: str) -> bool:
        """Cancel a running pipeline run
        
        Args:
            run_id (str): The ID of the run to cancel
        Returns:
            bool: True if the run was cancelled successfully, False otherwise
        """
        try:
            if self.use_api:
                return self._api_cancel_run(run_id)
            else:
                success = agent.cancel_run(run_id)
                if success:
                    print(f"🛑 Run {run_id} cancelled successfully")
                else:
                    print(f"❌ Failed to cancel run {run_id} (not found or not running)")
                return success
        except Exception as e:
            print(f"❌ Error cancelling run: {str(e)}")
            return False
    
    def _api_cancel_run(self, run_id: str) -> bool:
        """Cancel run via API
        
        Args:
            run_id (str): The ID of the run to cancel
        Returns:
            bool: True if the run was cancelled successfully, False otherwise
        """
        response = requests.post(f"{self.api_url}/api/runs/{run_id}/cancel")
        if response.status_code == requests.codes.ok:
            print(f"🛑 Run {run_id} cancelled successfully")
            return True
        else:
            error = response.json().get('error', 'Unknown error')
            print(f"❌ Error cancelling run: {error}")
            return False
    
    def delete_run(self, run_id: str, force: bool = False) -> bool:
        """Delete a specific run
        
        Args:
            run_id (str): The ID of the run to delete
            force (bool): Whether to force deletion without confirmation
        Returns:
            bool: True if the run was deleted successfully, False otherwise
        """
        try:
            if not force:
                confirm = input(f"⚠️  Are you sure you want to delete run {run_id}? (y/N): ")
                if confirm.lower() != 'y':
                    print("❌ Deletion cancelled")
                    return False
            
            if self.use_api:
                return self._api_delete_run(run_id)
            else:
                success = agent.delete_run(run_id)
                if success:
                    print(f"🗑️ Run {run_id} deleted successfully")
                else:
                    print(f"❌ Failed to delete run {run_id} (not found or currently running)")
                return success
        except Exception as e:
            print(f"❌ Error deleting run: {str(e)}")
            return False
    
    def _api_delete_run(self, run_id: str) -> bool:
        """Delete run via API

        Args:
            run_id (str): The ID of the run to delete
        Returns:
            bool: True if the run was deleted successfully, False otherwise
        """
        response = requests.delete(f"{self.api_url}/api/runs/{run_id}")
        if response.status_code == requests.codes.ok:
            print(f"🗑️ Run {run_id} deleted successfully")
            return True
        else:
            error = response.json().get('error', 'Unknown error')
            print(f"❌ Error deleting run: {error}")
            return False

def main():
    parser = argparse.ArgumentParser(description='CI/CD Agent CLI')
    parser.add_argument('--api-url', help='API URL for remote agent (optional)')
    
    subparsers = parser.add_subparsers(dest='command', help='Available commands')
    
    create_parser = subparsers.add_parser('create', help='Create a pipeline from JSON file')
    create_parser.add_argument('pipeline_file', help='Path to pipeline JSON file (use "-" for stdin)')
    
    run_parser = subparsers.add_parser('run', help='Run a pipeline by ID')
    run_parser.add_argument('pipeline_id', help='Pipeline ID to run')
    run_parser.add_argument('--foreground', action='store_true', help='Run in foreground (default: background)')
    
    create_run_parser = subparsers.add_parser('create-run', help='Create and run a pipeline')
    create_run_parser.add_argument('pipeline_file', help='Path to pipeline JSON file (use "-" for stdin)')
    create_run_parser.add_argument('--foreground', action='store_true', help='Run in foreground (default: background)')
    
    subparsers.add_parser('list', help='List all pipelines')
    
    status_parser = subparsers.add_parser('status', help='Get pipeline status')
    status_parser.add_argument('pipeline_id', help='Pipeline ID')
    
    cancel_parser = subparsers.add_parser('cancel', help='Cancel a running pipeline')
    cancel_parser.add_argument('pipeline_id', help='Pipeline ID to cancel')
    
    monitor_parser = subparsers.add_parser('monitor', help='Monitor a pipeline in real-time')
    monitor_parser.add_argument('pipeline_id', help='Pipeline ID to monitor')
    monitor_parser.add_argument('--interval', type=int, default=2, help='Refresh interval in seconds (default: 2)')
    
    delete_parser = subparsers.add_parser('delete', help='Delete a pipeline')
    delete_parser.add_argument('pipeline_id', help='Pipeline ID to delete')
    delete_parser.add_argument('--force', action='store_true', help='Force delete without confirmation')
    
    list_runs_parser = subparsers.add_parser('list-runs', help='List all runs')
    list_runs_parser.add_argument('--pipeline-id', help='Filter runs by pipeline ID')
    
    run_status_parser = subparsers.add_parser('run-status', help='Get run status')
    run_status_parser.add_argument('run_id', help='Run ID')
    
    cancel_run_parser = subparsers.add_parser('cancel-run', help='Cancel a running pipeline run')
    cancel_run_parser.add_argument('run_id', help='Run ID to cancel')
    
    delete_run_parser = subparsers.add_parser('delete-run', help='Delete a specific run')
    delete_run_parser.add_argument('run_id', help='Run ID to delete')
    delete_run_parser.add_argument('--force', action='store_true', help='Force delete without confirmation')
    
    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        return
    
    cli = CICDCli(api_url=args.api_url)
    
    if args.command == 'create':
        cli.create_pipeline(args.pipeline_file)
    elif args.command == 'run':
        cli.run_pipeline(args.pipeline_id, background=not args.foreground)
    elif args.command == 'create-run':
        cli.create_and_run(args.pipeline_file, background=not args.foreground)
    elif args.command == 'list':
        cli.list_pipelines()
    elif args.command == 'status':
        cli.get_pipeline_status(args.pipeline_id)
    elif args.command == 'cancel':
        cli.cancel_pipeline(args.pipeline_id)
    elif args.command == 'monitor':
        cli.monitor_pipeline(args.pipeline_id, args.interval)
    elif args.command == 'delete':
        if not args.force:
            confirm = input(f"Are you sure you want to delete pipeline {args.pipeline_id}? (y/N): ")
            if confirm.lower() not in ['y', 'yes']:
                print("❌ Delete cancelled")
                return
        cli.delete_pipeline(args.pipeline_id)
    elif args.command == 'list-runs':
        cli.list_runs(getattr(args, 'pipeline_id', None))
    elif args.command == 'run-status':
        cli.get_run_status(args.run_id)
    elif args.command == 'cancel-run':
        cli.cancel_run(args.run_id)
    elif args.command == 'delete-run':
        cli.delete_run(args.run_id, args.force)

if __name__ == '__main__':
    main() 
