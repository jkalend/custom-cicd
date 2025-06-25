import os
import sys
import logging
import json
import time
import datetime
import subprocess
import threading
from dataclasses import dataclass, field
from typing import Dict, List, Optional, Any
from enum import Enum
import uuid
import pickle
import fcntl #FIXME Doesn't work on windows

class PipelineStatus(Enum):
    PENDING = "pending"
    RUNNING = "running"
    SUCCESS = "success"
    FAILED = "failed"
    CANCELLED = "cancelled"

class StepStatus(Enum):
    PENDING = "pending"
    RUNNING = "running"
    SUCCESS = "success"
    FAILED = "failed"
    SKIPPED = "skipped"

def setup_logging():
    """Setup logging configuration"""
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s\n',
        handlers=[
            logging.StreamHandler(sys.stdout),
            logging.FileHandler('/app/logs/agent.log', mode='a')
        ]
    )
    return logging.getLogger(__name__)

logger = setup_logging()

@dataclass
class PipelineStep:
    name: str = field(default="")
    description: str = field(default="")
    command: str = field(default="")
    timeout: int = field(default=300)
    retry_count: int = field(default=0)
    continue_on_error: bool = field(default=False)
    status: StepStatus = field(default=StepStatus.PENDING)
    start_time: str = field(default="")
    end_time: str = field(default="")
    output: str = field(default="")
    error: str = field(default="")

    @classmethod
    def from_dict(cls, step_data: Dict[str, Any]) -> 'PipelineStep':
        """Create PipelineStep from dictionary data"""
        step = cls(
            name=step_data['name'],
            command=step_data['command'],
            description=step_data.get('description', ''),
            timeout=step_data.get('timeout', 300),
            retry_count=step_data.get('retry_count', 0),
            continue_on_error=step_data.get('continue_on_error', False)
        )
        
        # Restore runtime state if present
        if 'status' in step_data:
            step.status = StepStatus(step_data['status'])
        if 'start_time' in step_data:
            step.start_time = step_data['start_time']
        if 'end_time' in step_data:
            step.end_time = step_data['end_time']
        if 'output' in step_data:
            step.output = step_data['output']
        if 'error' in step_data:
            step.error = step_data['error']
        
        return step

    def to_dict(self) -> Dict[str, Any]:
        """Convert PipelineStep to dictionary"""
        return {
            'name': self.name,
            'description': self.description,
            'command': self.command,
            'timeout': self.timeout,
            'retry_count': self.retry_count,
            'continue_on_error': self.continue_on_error,
            'status': self.status.value,
            'start_time': self.start_time,
            'end_time': self.end_time,
            'output': self.output,
            'error': self.error
        }

@dataclass
class PipelineRun:
    id: str = field(default_factory=lambda: str(uuid.uuid4()))
    pipeline_id: str = field(default="")
    name: str = field(default="")
    version: str = field(default="1.0.0")
    description: str = field(default="")
    variables: Dict[str, Any] = field(default_factory=dict)
    steps: List[PipelineStep] = field(default_factory=list)
    status: PipelineStatus = field(default=PipelineStatus.PENDING)
    created_at: str = field(default_factory=lambda: datetime.datetime.now().isoformat())
    started_at: Optional[str] = None
    finished_at: Optional[str] = None
    total_duration: Optional[float] = None

    @classmethod
    def from_config(cls, pipeline_id: str, pipeline_config: Dict[str, Any]) -> 'PipelineRun':
        """Create PipelineRun from pipeline configuration"""
        return cls(
            pipeline_id=pipeline_id,
            name=pipeline_config['name'],
            version=pipeline_config.get('version', '1.0.0'),
            description=pipeline_config.get('description', ''),
            variables=pipeline_config.get('variables', {}),
            steps=[PipelineStep.from_dict(step) for step in pipeline_config['steps']]
        )
    
    @classmethod
    def from_dict(cls, run_data: Dict[str, Any]) -> 'PipelineRun':
        """Create PipelineRun from dictionary data (for deserialization)"""
        return cls(
            id=run_data['id'],
            pipeline_id=run_data['pipeline_id'],
            name=run_data['name'],
            version=run_data['version'],
            description=run_data['description'],
            variables=run_data['variables'],
            status=PipelineStatus(run_data['status']),
            created_at=run_data['created_at'],
            started_at=run_data['started_at'],
            finished_at=run_data['finished_at'],
            total_duration=run_data['total_duration'],
            steps=[PipelineStep.from_dict(step_data) for step_data in run_data['steps']]
        )

    def to_dict(self) -> Dict[str, Any]:
        """Convert PipelineRun to dictionary"""
        return {
            'id': self.id,
            'pipeline_id': self.pipeline_id,
            'name': self.name,
            'version': self.version,
            'description': self.description,
            'variables': self.variables,
            'status': self.status.value,
            'created_at': self.created_at,
            'started_at': self.started_at,
            'finished_at': self.finished_at,
            'total_duration': self.total_duration,
            'steps': [step.to_dict() for step in self.steps]
        }

    def to_summary_dict(self) -> Dict[str, Any]:
        """Convert PipelineRun to summary dictionary (without steps)"""
        return {
            'id': self.id,
            'pipeline_id': self.pipeline_id,
            'name': self.name,
            'status': self.status.value,
            'created_at': self.created_at,
            'started_at': self.started_at,
            'finished_at': self.finished_at,
            'total_duration': self.total_duration
        }

class CICDAgent:
    def __init__(self, data_file='/app/data/agent_data.pkl'):
        self.data_file = data_file
        self.pipelines = {}
        self.runs = {}
        self.run_history = []
        
        os.makedirs(os.path.dirname(self.data_file), exist_ok=True)
        
        self._load_data()
        self._cleanup_data()

    # def __new__(cls, *args, **kwargs):
    #     if not hasattr(cls, 'instance'):
    #         cls.instance = super(CICDAgent, cls).__new__(cls)
    #     return cls.instance
    
    def _load_data(self):
        """Load persisted data from file"""
        try:
            if os.path.exists(self.data_file):
                with open(self.data_file, 'rb') as f:
                    fcntl.flock(f.fileno(), fcntl.LOCK_SH)
                    data = pickle.load(f)
                    fcntl.flock(f.fileno(), fcntl.LOCK_UN)
                    
                    self.pipelines = data.get('pipelines', {})
                    
                    self.runs = {}
                    for run_id, run_data in data.get('runs', {}).items():
                        run = PipelineRun.from_dict(run_data)
                        self.runs[run_id] = run
                    
                    self.run_history = data.get('run_history', [])
                    
                    logger.info(f"Loaded {len(self.pipelines)} pipelines, {len(self.runs)} runs, {len(self.run_history)} run history")
        except Exception as e:
            logger.warning(f"Failed to load persisted data: {str(e)}")
            # Try to recover by clearing corrupted data
            try:
                if os.path.exists(self.data_file):
                    backup_file = self.data_file + '.backup'
                    os.rename(self.data_file, backup_file)
                    logger.info(f"Moved corrupted data file to {backup_file}")
            except:
                pass
            # Initialize with empty data if loading fails
            self.pipelines = {}
            self.runs = {}
            self.run_history = []
    
    def _save_data(self):
        """Save current data to file"""
        try:
            serializable_pipelines = {}
            for pid, pipeline in self.pipelines.items():
                serializable_pipeline = pipeline.copy()
                serializable_pipelines[pid] = serializable_pipeline
            
            serializable_runs = {}
            for run_id, run in self.runs.items():
                serializable_runs[run_id] = run.to_dict()
            
            data = {
                'pipelines': serializable_pipelines,
                'runs': serializable_runs,
                'run_history': self.run_history
            }
            
            with open(self.data_file, 'wb') as f:
                fcntl.flock(f.fileno(), fcntl.LOCK_EX)
                pickle.dump(data, f)
                fcntl.flock(f.fileno(), fcntl.LOCK_UN)
                
            logger.debug(f"Saved {len(serializable_pipelines)} pipelines to {self.data_file}")
            
        except Exception as e:
            logger.error(f"Failed to save data: {str(e)}")
    
    def _cleanup_data(self):
        """Clean up any data inconsistencies after loading"""
        try:
            cleaned_run_history = []
            for run in self.run_history:
                cleaned_run = run.copy()
                
                if hasattr(cleaned_run.get('status'), 'value'):
                    cleaned_run['status'] = cleaned_run['status'].value
                elif cleaned_run.get('status') is None:
                    cleaned_run['status'] = 'unknown'
                
                cleaned_run_history.append(cleaned_run)
            
            self.run_history = cleaned_run_history
            logger.debug("Data cleanup completed")
            
        except Exception as e:
            logger.warning(f"Error during data cleanup: {str(e)}")

    def _get_status_value(self, status_obj) -> str:
        """Safely extract status value from status object"""
        if hasattr(status_obj, 'value'):
            return status_obj.value
        elif status_obj is None:
            return 'unknown'
        else:
            return str(status_obj)

    def load_pipeline_from_file(self, pipeline_file: str) -> str:
        """Load pipeline from JSON file and return pipeline ID"""
        try:
            logger.info(f"Loading pipeline from {pipeline_file}")
            with open(pipeline_file, 'r') as file:
                pipeline_config = json.load(file)
                
            return self.create_pipeline(pipeline_config)
        except Exception as e:
            logger.error(f"Failed to load pipeline from {pipeline_file}: {str(e)}")
            raise

    def create_pipeline(self, pipeline_config: Dict[str, Any]) -> str:
        """Create a new pipeline definition and return its ID"""
        pipeline_id = str(uuid.uuid4())
        
        pipeline = {
            'id': pipeline_id,
            'name': pipeline_config['name'],
            'version': pipeline_config.get('version', '1.0.0'),
            'description': pipeline_config.get('description', ''),
            'variables': pipeline_config.get('variables', {}),
            'steps': pipeline_config['steps'],
            'created_at': datetime.datetime.now().isoformat(),
            'updated_at': datetime.datetime.now().isoformat()
        }
        
        self.pipelines[pipeline_id] = pipeline
        self._save_data()
        logger.info(f"Created pipeline definition {pipeline['name']} with ID {pipeline_id}")
        return pipeline_id
    
    def create_run(self, pipeline_id: str) -> str:
        """Create a new run for an existing pipeline"""
        if pipeline_id not in self.pipelines:
            raise ValueError(f"Pipeline {pipeline_id} not found")
        
        pipeline_def = self.pipelines[pipeline_id]
        run = PipelineRun.from_config(pipeline_id, pipeline_def)
            
        self.runs[run.id] = run
        self._save_data()
        logger.info(f"Created run {run.id} for pipeline {pipeline_def['name']}")
        return run.id

    def run_pipeline(self, pipeline_id: str, background: bool = True) -> str:
        """Create and run a new run for the pipeline. Returns run ID."""
        run_id = self.create_run(pipeline_id)
        
        if background:
            thread = threading.Thread(target=self._execute_run, args=(run_id,))
            thread.daemon = True
            thread.start()
            return run_id
        else:
            success = self._execute_run(run_id)
            return run_id if success else None
    
    def _execute_run(self, run_id: str) -> bool:
        """Execute a pipeline run"""
        if run_id not in self.runs:
            logger.error(f"Run {run_id} not found")
            return False
            
        run = self.runs[run_id]
        
        try:
            logger.info(f"Starting run {run_id} for pipeline {run.name}")
            run.status = PipelineStatus.RUNNING
            run.started_at = datetime.datetime.now().isoformat()
            
            start_time = time.time()
            
            for step in run.steps:
                if run.status == PipelineStatus.CANCELLED:
                    break
                    
                success = self._execute_step(step, run.variables)
                
                if not success and not step.continue_on_error:
                    run.status = PipelineStatus.FAILED
                    break
            else:
                run.status = PipelineStatus.SUCCESS
            
            end_time = time.time()
            run.finished_at = datetime.datetime.now().isoformat()
            run.total_duration = end_time - start_time
            
            self._move_run_to_history(run)
            self._save_data()
            logger.info(f"Run {run_id} finished with status: {run.status.value}")
            return run.status == PipelineStatus.SUCCESS
            
        except Exception as e:
            logger.error(f"Run {run_id} failed with exception: {str(e)}")
            run.status = PipelineStatus.FAILED
            run.finished_at = datetime.datetime.now().isoformat()
            self._move_run_to_history(run)
            self._save_data()
            return False
    
    def _move_run_to_history(self, run: PipelineRun):
        """Move a completed run to history"""
        history_run = run.to_dict()
        
        self.run_history.append(history_run)
        
        if run.id in self.runs:
            del self.runs[run.id]
    
    def _execute_step(self, step: PipelineStep, variables: Dict[str, str]) -> bool:
        """Execute a single pipeline step"""
        logger.info(f"Executing step: {step.name}")
        step.status = StepStatus.RUNNING
        step.start_time = datetime.datetime.now().isoformat()
        
        try:
            command = step.command
            for key, value in variables.items():
                command = command.replace(f"${{{key}}}", str(value))
                command = command.replace(f"${key}", str(value))
            
            logger.info(f"Running command: {command}")
            
            for attempt in range(step.retry_count + 1):
                try:
                    result = subprocess.run(
                        command,
                        shell=True,
                        check=True,
                        timeout=step.timeout,
                        capture_output=True,
                        text=True
                    )
                    
                    step.output = result.stdout
                    step.status = StepStatus.SUCCESS
                    step.end_time = datetime.datetime.now().isoformat()
                    logger.info(f"Step {step.name} completed successfully")
                    return True
                    
                except subprocess.CalledProcessError as e:
                    step.error = e.stderr
                    if attempt < step.retry_count:
                        logger.warning(f"Step {step.name} failed (attempt {attempt + 1}), retrying...")
                        time.sleep(2 ** attempt)  # Exponential backoff
                    else:
                        step.status = StepStatus.FAILED
                        step.end_time = datetime.datetime.now().isoformat()
                        logger.error(f"Step {step.name} failed after {attempt + 1} attempts: {e.stderr}")
                        return False
                        
                except subprocess.TimeoutExpired:
                    step.error = f"Command timed out after {step.timeout} seconds"
                    step.status = StepStatus.FAILED
                    step.end_time = datetime.datetime.now().isoformat()
                    logger.error(f"Step {step.name} timed out")
                    return False
                    
        except Exception as e:
            step.error = str(e)
            step.status = StepStatus.FAILED
            step.end_time = datetime.datetime.now().isoformat()
            logger.error(f"Step {step.name} failed with exception: {str(e)}")
            return False

    def cancel_run(self, run_id: str) -> bool:
        """Cancel a running pipeline run"""
        if run_id in self.runs:
            self.runs[run_id].status = PipelineStatus.CANCELLED
            logger.info(f"Run {run_id} cancelled")
            self._save_data()
            return True
        return False
    
    def cancel_pipeline(self, pipeline_id: str) -> bool:
        """Cancel all active runs for a pipeline"""
        cancelled = False
        for run_id, run in self.runs.items():
            if run.pipeline_id == pipeline_id:
                run.status = PipelineStatus.CANCELLED
                cancelled = True
        
        if cancelled:
            logger.info(f"Cancelled all runs for pipeline {pipeline_id}")
            self._save_data()
            
        return cancelled

    def get_pipeline_status(self, pipeline_id: str) -> Optional[Dict[str, Any]]:
        """Get pipeline definition status"""
        if pipeline_id in self.pipelines:
            pipeline = self.pipelines[pipeline_id]
            
            # Get latest run status for this pipeline
            latest_run = None
            for run in self.runs.values():
                if run.pipeline_id == pipeline_id:
                    if latest_run is None or run.created_at > latest_run.created_at:
                        latest_run = run
            
            # Check run history for latest completed run if no active run
            if latest_run is None:
                for run in sorted(self.run_history, key=lambda x: x.get('created_at', ''), reverse=True):
                    if run.get('pipeline_id') == pipeline_id:
                        return {
                            'id': pipeline['id'],
                            'name': pipeline['name'],
                            'description': pipeline.get('description', ''),
                            'status': run.get('status', 'unknown'),
                            'created_at': pipeline['created_at'],
                            'last_run_at': run.get('started_at'),
                            'last_finished_at': run.get('finished_at'),
                            'total_duration': run.get('total_duration'),
                            'steps': run.get('steps', [])
                        }
            
            # Return pipeline definition with latest run info
            if latest_run:
                return {
                    'id': pipeline['id'],
                    'name': pipeline['name'],
                    'description': pipeline.get('description', ''),
                    'status': latest_run.status.value,
                    'created_at': pipeline['created_at'],
                    'last_run_at': latest_run.started_at,
                    'last_finished_at': latest_run.finished_at,
                    'total_duration': latest_run.total_duration,
                    'steps': [step.to_dict() for step in latest_run.steps]
                }
            else:
                # Pipeline exists but never run
                return {
                    'id': pipeline['id'],
                    'name': pipeline['name'],
                    'description': pipeline.get('description', ''),
                    'status': 'never_run',
                    'created_at': pipeline['created_at'],
                    'last_run_at': None,
                    'last_finished_at': None,
                    'total_duration': None,
                    'steps': []
                }
        
        return None

    def list_pipelines(self) -> List[Dict[str, Any]]:
        """List all pipeline definitions"""
        all_pipelines = []
        
        for pipeline in self.pipelines.values():
            latest_run_status = 'never_run'
            latest_run_time = None
            active_runs = 0
            
            for run in self.runs.values():
                if run.pipeline_id == pipeline['id']:
                    active_runs += 1
                    latest_run_status = run.status.value
                    if latest_run_time is None or run.created_at > latest_run_time:
                        latest_run_time = run.created_at
            
            if active_runs == 0:
                for run in sorted(self.run_history, key=lambda x: x.get('created_at', ''), reverse=True):
                    if run.get('pipeline_id') == pipeline['id']:
                        latest_run_status = run.get('status', 'unknown')
                        latest_run_time = run.get('created_at')
                        break
            
            all_pipelines.append({
                'id': pipeline['id'],
                'name': pipeline['name'],
                'description': pipeline.get('description', ''),
                'status': latest_run_status,
                'created_at': pipeline['created_at'],
                'last_run_at': latest_run_time,
                'active_runs': active_runs,
                'total_runs': len([r for r in self.run_history if r.get('pipeline_id') == pipeline['id']]) + active_runs
            })
        
        return sorted(all_pipelines, key=lambda x: x['created_at'], reverse=True)
    
    def list_runs(self, pipeline_id: str = None) -> List[Dict[str, Any]]:
        """List all runs, optionally filtered by pipeline ID"""
        all_runs = []
        
        for run in self.runs.values():
            if pipeline_id is None or run.pipeline_id == pipeline_id:
                all_runs.append(run.to_summary_dict())
        
        for run in self.run_history:
            if pipeline_id is None or run['pipeline_id'] == pipeline_id:
                all_runs.append({
                    'id': run['id'],
                    'pipeline_id': run['pipeline_id'],
                    'name': run['name'],
                    'status': self._get_status_value(run['status']),
                    'created_at': run['created_at'],
                    'started_at': run['started_at'],
                    'finished_at': run['finished_at'],
                    'total_duration': run.get('total_duration')
                })
        
        return sorted(all_runs, key=lambda x: x['created_at'], reverse=True)
    
    def get_run_status(self, run_id: str) -> Optional[Dict[str, Any]]:
        """Get run status and details"""
        # Check active runs
        if run_id in self.runs:
            run = self.runs[run_id]
            return {
                'id': run.id,
                'pipeline_id': run.pipeline_id,
                'name': run.name,
                'status': run.status.value,
                'created_at': run.created_at,
                'started_at': run.started_at,
                'finished_at': run.finished_at,
                'total_duration': run.total_duration,
                'steps': [step.to_dict() for step in run.steps]
            }
        
        # Check history
        for hist_run in self.run_history:
            if hist_run['id'] == run_id:
                return {
                    'id': hist_run['id'],
                    'pipeline_id': hist_run['pipeline_id'],
                    'name': hist_run['name'],
                    'status': self._get_status_value(hist_run['status']),
                    'created_at': hist_run['created_at'],
                    'started_at': hist_run['started_at'],
                    'finished_at': hist_run['finished_at'],
                    'total_duration': hist_run.get('total_duration'),
                    'steps': hist_run.get('steps', [])
                }
        
        return None

    def delete_run(self, run_id: str) -> bool:
        """Delete a specific run by ID"""
        # Check if run is currently active
        if run_id in self.runs:
            run = self.runs[run_id]
            if run.status == PipelineStatus.RUNNING:
                logger.error(f"Cannot delete run {run_id}: currently running")
                return False
            
            del self.runs[run_id]
            self._save_data()
            logger.info(f"Deleted active run {run_id}")
            return True
        
        # Check if run is in history
        for i, hist_run in enumerate(self.run_history):
            if hist_run['id'] == run_id:
                del self.run_history[i]
                self._save_data()
                logger.info(f"Deleted historical run {run_id}")
                return True
        
        logger.error(f"Run {run_id} not found")
        return False

    def delete_pipeline(self, pipeline_id: str) -> bool:
        """Delete a pipeline definition by ID"""
        active_runs = [run_id for run_id, run in self.runs.items() 
                      if run.pipeline_id == pipeline_id and run.status == PipelineStatus.RUNNING]
        
        if active_runs:
            logger.error(f"Cannot delete pipeline {pipeline_id}: has active runs {active_runs}")
            return False
        
        if pipeline_id in self.pipelines:
            pipeline_name = self.pipelines[pipeline_id]['name']
            del self.pipelines[pipeline_id]
            self._save_data()
            logger.info(f"Deleted pipeline definition {pipeline_name} with ID {pipeline_id}")
            return True
        
        logger.warning(f"Pipeline {pipeline_id} not found")
        return False

agent = CICDAgent() 
