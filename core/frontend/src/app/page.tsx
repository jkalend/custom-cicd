'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { 
  RocketLaunchIcon, 
  ClipboardDocumentListIcon,
  PlayIcon,
  StopIcon,
  TrashIcon,
  ArrowPathIcon,
  PlusIcon,
  DocumentTextIcon,
  EyeIcon
} from '@heroicons/react/24/outline';

import { Pipeline, PipelineRun, CreatePipelineRequest } from '@/types/api';
import { apiClient } from '@/lib/api';
import { formatDate, formatDuration, formatRelativeTime, defaultPipelineTemplate } from '@/lib/utils';
import { StatusBadge } from '@/components/ui/status-badge';
import { LogViewer, useLogs } from '@/components/ui/log-viewer';
import { Pagination } from '@/components/ui/pagination';

export default function Dashboard() {
  const [pipelines, setPipelines] = useState<Pipeline[]>([]);
  const [runs, setRuns] = useState<PipelineRun[]>([]);
  const [pipelineJson, setPipelineJson] = useState(JSON.stringify(defaultPipelineTemplate, null, 2));
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pipelinePage, setPipelinePage] = useState(1);
  const [runsPage, setRunsPage] = useState(1);
  const router = useRouter();
  
  const { logs, addLog, clearLogs } = useLogs();
  
  // Pagination settings
  const ITEMS_PER_PAGE = 5;
  
  // Calculate paginated data
  const paginatedPipelines = pipelines.slice(
    (pipelinePage - 1) * ITEMS_PER_PAGE,
    pipelinePage * ITEMS_PER_PAGE
  );
  
  const paginatedRuns = runs.slice(
    (runsPage - 1) * ITEMS_PER_PAGE,
    runsPage * ITEMS_PER_PAGE
  );

  // Auto-refresh every 5 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      loadData();
    }, 5000);

    // Load initial data
    loadData();

    return () => clearInterval(interval);
  }, []);

  const loadData = async () => {
    try {
      const [pipelinesData, runsData] = await Promise.all([
        apiClient.listPipelines(),
        apiClient.listRuns()
      ]);
      
      // Ensure we have arrays
      setPipelines(Array.isArray(pipelinesData) ? pipelinesData : []);
      setRuns(Array.isArray(runsData) ? runsData : []);
      setError(null); // Clear any previous errors
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load data';
      setError(message);
      addLog(`❌ Error loading data: ${message}`, 'error');
      // Set empty arrays on error to prevent map errors
      setPipelines([]);
      setRuns([]);
    }
  };

  const createPipeline = async () => {
    try {
      setLoading(true);
      const pipeline: CreatePipelineRequest = JSON.parse(pipelineJson);
      const result = await apiClient.createPipeline(pipeline);
      addLog(`✅ Pipeline created: ${result.pipeline_id}`, 'success');
      loadData();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create pipeline';
      addLog(`❌ Failed to create pipeline: ${message}`, 'error');
    } finally {
      setLoading(false);
    }
  };

  const createAndRunPipeline = async () => {
    try {
      setLoading(true);
      const pipeline: CreatePipelineRequest = JSON.parse(pipelineJson);
      const result = await apiClient.createAndRunPipeline(pipeline);
      addLog(`🚀 Pipeline created and started: ${result.pipeline_id}`, 'success');
      loadData();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create and run pipeline';
      addLog(`❌ Failed to create and run pipeline: ${message}`, 'error');
    } finally {
      setLoading(false);
    }
  };

  const runPipeline = async (pipelineId: string) => {
    try {
      const result = await apiClient.runPipeline(pipelineId);
      addLog(`🚀 Pipeline started: ${pipelineId}`, 'info');
      loadData();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to start pipeline';
      addLog(`❌ Failed to start pipeline: ${message}`, 'error');
    }
  };

  const cancelPipeline = async (pipelineId: string) => {
    try {
      await apiClient.cancelPipeline(pipelineId);
      addLog(`🛑 Pipeline cancelled: ${pipelineId}`, 'warning');
      loadData();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to cancel pipeline';
      addLog(`❌ Failed to cancel pipeline: ${message}`, 'error');
    }
  };

  const deletePipeline = async (pipelineId: string) => {
    if (!confirm('Are you sure you want to delete this pipeline? This action cannot be undone.')) {
      return;
    }
    try {
      await apiClient.deletePipeline(pipelineId);
      addLog(`🗑️ Pipeline deleted: ${pipelineId}`, 'warning');
      loadData();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to delete pipeline';
      addLog(`❌ Failed to delete pipeline: ${message}`, 'error');
    }
  };

  const showPipelineDetails = (pipelineId: string) => {
    // Navigate to the pipeline details page
    router.push(`/pipeline/${pipelineId}`);
  };



  const cancelRun = async (runId: string) => {
    try {
      await apiClient.cancelRun(runId);
      addLog(`🛑 Run cancelled: ${runId}`, 'warning');
      loadData();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to cancel run';
      addLog(`❌ Failed to cancel run: ${message}`, 'error');
    }
  };

  const deleteRun = async (runId: string) => {
    if (!confirm('Are you sure you want to delete this run? This action cannot be undone.')) {
      return;
    }
    try {
      await apiClient.deleteRun(runId);
      addLog(`🗑️ Run deleted: ${runId}`, 'warning');
      loadData();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to delete run';
      addLog(`❌ Failed to delete run: ${message}`, 'error');
    }
  };

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="bg-gray-900 text-white rounded-lg p-6 mb-6">
          <div className="flex items-center gap-3 mb-2">
            <RocketLaunchIcon className="w-8 h-8" />
            <h1 className="text-3xl font-bold">CI/CD Agent Dashboard</h1>
          </div>
          <p className="text-gray-300">Manage and monitor your CI/CD pipelines</p>
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded mb-6">
            {error}
          </div>
        )}

        {/* Create Pipeline */}
        <div className="bg-white rounded-lg shadow p-6 mb-6">
          <div className="flex items-center gap-2 mb-4">
            <ClipboardDocumentListIcon className="w-5 h-5 text-gray-700" />
            <h2 className="text-xl font-semibold text-gray-900">Create Pipeline</h2>
          </div>
          
          <textarea
            value={pipelineJson}
            onChange={(e) => setPipelineJson(e.target.value)}
            className="w-full h-64 p-3 border border-gray-300 rounded-lg font-mono text-sm text-gray-900"
            placeholder="Paste your pipeline JSON here..."
          />
          
          <div className="flex gap-3 mt-4">
            <button
              onClick={createPipeline}
              disabled={loading}
              className="flex items-center gap-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white px-4 py-2 rounded-lg"
            >
              <PlusIcon className="w-4 h-4" />
              Create Pipeline
            </button>
            <button
              onClick={createAndRunPipeline}
              disabled={loading}
              className="flex items-center gap-2 bg-green-600 hover:bg-green-700 disabled:bg-green-400 text-white px-4 py-2 rounded-lg"
            >
              <PlayIcon className="w-4 h-4" />
              Create & Run
            </button>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          {/* Pipelines */}
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold text-gray-900">📊 Pipelines ({pipelines.length})</h2>
              <button
                onClick={loadData}
                className="flex items-center gap-2 text-blue-600 hover:text-blue-700"
              >
                <ArrowPathIcon className="w-4 h-4" />
                Refresh
              </button>
            </div>
            
            {pipelines.length === 0 ? (
              <p className="text-gray-700">No pipelines found.</p>
            ) : (
              <>
                <div className="space-y-4 mb-4">
                  {paginatedPipelines.map((pipeline) => (
                    <div key={pipeline.id} className="border-l-4 border-blue-500 pl-4 p-3 bg-gray-50 rounded">
                      <div className="flex items-center justify-between mb-2">
                        <h3 className="font-medium text-gray-900">{pipeline.name}</h3>
                        <StatusBadge status={pipeline.status} />
                      </div>
                      
                      <div className="text-sm text-gray-800 space-y-1">
                        <p><strong className="text-gray-900">ID:</strong> {pipeline.id}</p>
                        <p><strong className="text-gray-900">Description:</strong> {pipeline.description || 'No description'}</p>
                        <p><strong className="text-gray-900">Created:</strong> {formatRelativeTime(pipeline.created_at)}</p>
                        <p><strong className="text-gray-900">Active Runs:</strong> {pipeline.active_runs || 0}</p>
                        <p><strong className="text-gray-900">Total Runs:</strong> {pipeline.total_runs || 0}</p>
                      </div>
                      
                      <div className="flex flex-wrap gap-2 mt-3">
                        <button
                          onClick={() => runPipeline(pipeline.id)}
                          className="text-xs bg-green-600 hover:bg-green-700 text-white px-2 py-1 rounded"
                        >
                          ▶️ Run
                        </button>
                        <button
                          onClick={() => showPipelineDetails(pipeline.id)}
                          className="text-xs bg-yellow-600 hover:bg-yellow-700 text-white px-2 py-1 rounded"
                        >
                          👁️ View Details & Runs
                        </button>
                        <button
                          onClick={() => cancelPipeline(pipeline.id)}
                          className="text-xs bg-gray-600 hover:bg-gray-700 text-white px-2 py-1 rounded"
                        >
                          🛑 Cancel
                        </button>
                        <button
                          onClick={() => deletePipeline(pipeline.id)}
                          className="text-xs bg-red-600 hover:bg-red-700 text-white px-2 py-1 rounded"
                        >
                          🗑️ Delete
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
                
                <Pagination
                  currentPage={pipelinePage}
                  totalItems={pipelines.length}
                  itemsPerPage={ITEMS_PER_PAGE}
                  onPageChange={setPipelinePage}
                />
              </>
            )}
          </div>

          {/* Runs */}
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold text-gray-900">🏃 Pipeline Runs ({runs.length})</h2>
              <button
                onClick={loadData}
                className="flex items-center gap-2 text-blue-600 hover:text-blue-700"
              >
                <ArrowPathIcon className="w-4 h-4" />
                Refresh
              </button>
            </div>
            
            {runs.length === 0 ? (
              <p className="text-gray-700">No runs found.</p>
            ) : (
              <>
                <div className="space-y-4 mb-4">
                  {paginatedRuns.map((run) => (
                    <div key={run.id} className="border-l-4 border-green-500 pl-4 p-3 bg-gray-50 rounded">
                      <div className="flex items-center justify-between mb-2">
                        <button
                          onClick={() => router.push(`/run/${run.id}`)}
                          className="font-medium text-sm text-gray-900 hover:text-blue-600 text-left"
                        >
                          {run.name}
                        </button>
                        <StatusBadge status={run.status} />
                      </div>
                      
                      <div className="text-xs text-gray-800 space-y-1">
                        <p><strong className="text-gray-900">Run ID:</strong> {run.id.slice(0, 8)}...</p>
                        <p><strong className="text-gray-900">Pipeline:</strong> {run.pipeline_id.slice(0, 8)}...</p>
                        <p><strong className="text-gray-900">Created:</strong> {formatRelativeTime(run.created_at)}</p>
                        {run.total_duration && (
                          <p><strong className="text-gray-900">Duration:</strong> {formatDuration(run.total_duration)}</p>
                        )}
                      </div>
                      
                      <div className="flex flex-wrap gap-2 mt-2">
                        <button
                          onClick={() => router.push(`/run/${run.id}`)}
                          className="text-xs bg-blue-600 hover:bg-blue-700 text-white px-2 py-1 rounded"
                        >
                          👁️ View Details
                        </button>
                        {run.status === 'running' && (
                          <button
                            onClick={() => cancelRun(run.id)}
                            className="text-xs bg-gray-600 hover:bg-gray-700 text-white px-2 py-1 rounded"
                          >
                            🛑 Cancel
                          </button>
                        )}
                        <button
                          onClick={() => deleteRun(run.id)}
                          className="text-xs bg-red-600 hover:bg-red-700 text-white px-2 py-1 rounded"
                        >
                          🗑️ Delete
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
                
                <Pagination
                  currentPage={runsPage}
                  totalItems={runs.length}
                  itemsPerPage={ITEMS_PER_PAGE}
                  onPageChange={setRunsPage}
                />
              </>
            )}
          </div>
        </div>

        {/* Logs */}
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-semibold text-gray-900">📝 Logs</h2>
            <button
              onClick={clearLogs}
              className="flex items-center gap-2 text-yellow-600 hover:text-yellow-700"
            >
              <TrashIcon className="w-4 h-4" />
              Clear Logs
            </button>
          </div>
          
          <LogViewer logs={logs} />
        </div>
      </div>
    </div>
  );
}
