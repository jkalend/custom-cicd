'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { 
  ArrowLeftIcon,
  PlayIcon,
  StopIcon,
  TrashIcon,
  ArrowPathIcon,
  ClockIcon,
  CalendarIcon,
  DocumentTextIcon,
  CogIcon,
  ChevronRightIcon,
  ChevronDownIcon,
  ChevronUpIcon
} from '@heroicons/react/24/outline';

import { Pipeline, PipelineRun } from '@/types/api';
import { apiClient } from '@/lib/api';
import { formatDate, formatDuration, formatRelativeTime } from '@/lib/utils';
import { StatusBadge } from '@/components/ui/status-badge';
import { LogViewer, useLogs } from '@/components/ui/log-viewer';
import { Pagination } from '@/components/ui/pagination';

interface PipelineDetailsPageProps {
  params: { id: string };
}

export default function PipelineDetailsPage({ params }: PipelineDetailsPageProps) {
  const [pipeline, setPipeline] = useState<Pipeline | null>(null);
  const [runs, setRuns] = useState<PipelineRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [runsPage, setRunsPage] = useState(1);
  const [stepsExpanded, setStepsExpanded] = useState(false);
  const router = useRouter();
  const { logs, addLog, clearLogs } = useLogs();
  
  // Pagination settings
  const ITEMS_PER_PAGE = 5;
  
  // Calculate paginated runs
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
  }, [params.id]);

  const loadData = async () => {
    try {
      const [pipelineData, runsData] = await Promise.all([
        apiClient.getPipeline(params.id),
        apiClient.listRuns(params.id)
      ]);
      
      setPipeline(pipelineData);
      setRuns(Array.isArray(runsData) ? runsData : []);
      setError(null);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load pipeline data';
      setError(message);
      addLog(`❌ Error loading pipeline data: ${message}`, 'error');
    } finally {
      setLoading(false);
    }
  };

  const runPipeline = async () => {
    if (!pipeline) return;
    try {
      await apiClient.runPipeline(pipeline.id);
      addLog(`🚀 Pipeline started: ${pipeline.name}`, 'success');
      loadData();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to start pipeline';
      addLog(`❌ Failed to start pipeline: ${message}`, 'error');
    }
  };

  const cancelPipeline = async () => {
    if (!pipeline) return;
    try {
      await apiClient.cancelPipeline(pipeline.id);
      addLog(`🛑 Pipeline cancelled: ${pipeline.name}`, 'warning');
      loadData();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to cancel pipeline';
      addLog(`❌ Failed to cancel pipeline: ${message}`, 'error');
    }
  };

  const deletePipeline = async () => {
    if (!pipeline) return;
    if (!confirm(`Are you sure you want to delete pipeline "${pipeline.name}"? This action cannot be undone.`)) {
      return;
    }
    try {
      await apiClient.deletePipeline(pipeline.id);
      addLog(`🗑️ Pipeline deleted: ${pipeline.name}`, 'warning');
      router.push('/');
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to delete pipeline';
      addLog(`❌ Failed to delete pipeline: ${message}`, 'error');
    }
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

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <ArrowPathIcon className="w-8 h-8 animate-spin mx-auto mb-4 text-blue-600" />
          <p className="text-gray-600">Loading pipeline...</p>
        </div>
      </div>
    );
  }

  if (error || !pipeline) {
    return (
      <div className="min-h-screen bg-gray-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded mb-6">
            {error || 'Pipeline not found'}
          </div>
          <Link
            href="/"
            className="flex items-center gap-2 text-blue-600 hover:text-blue-700"
          >
            <ArrowLeftIcon className="w-4 h-4" />
            Back to Dashboard
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="bg-white rounded-lg shadow p-6 mb-6">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-4">
              <Link
                href="/"
                className="flex items-center gap-2 text-blue-600 hover:text-blue-700"
              >
                <ArrowLeftIcon className="w-4 h-4" />
                Back to Dashboard
              </Link>
              <ChevronRightIcon className="w-4 h-4 text-gray-400" />
              <h1 className="text-2xl font-bold text-gray-900">{pipeline.name}</h1>
            </div>
            <StatusBadge status={pipeline.status} />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
            <div className="flex items-center gap-2 text-gray-600">
              <DocumentTextIcon className="w-4 h-4" />
              <span className="text-sm">Version: {pipeline.version}</span>
            </div>
            <div className="flex items-center gap-2 text-gray-600">
              <CalendarIcon className="w-4 h-4" />
              <span className="text-sm">Created: {formatRelativeTime(pipeline.created_at)}</span>
            </div>
            <div className="flex items-center gap-2 text-gray-600">
              <ClockIcon className="w-4 h-4" />
              <span className="text-sm">
                Last run: {pipeline.last_run_at ? formatRelativeTime(pipeline.last_run_at) : 'Never'}
              </span>
            </div>
          </div>

          {pipeline.description && (
            <p className="text-gray-700 mb-6">{pipeline.description}</p>
          )}

          {/* Action Buttons */}
          <div className="flex flex-wrap gap-3">
            <button
              onClick={runPipeline}
              className="flex items-center gap-2 bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-lg"
            >
              <PlayIcon className="w-4 h-4" />
              Run Pipeline
            </button>
            <button
              onClick={cancelPipeline}
              className="flex items-center gap-2 bg-gray-600 hover:bg-gray-700 text-white px-4 py-2 rounded-lg"
            >
              <StopIcon className="w-4 h-4" />
              Cancel All
            </button>
            <button
              onClick={() => loadData()}
              className="flex items-center gap-2 bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg"
            >
              <ArrowPathIcon className="w-4 h-4" />
              Refresh
            </button>
            <button
              onClick={deletePipeline}
              className="flex items-center gap-2 bg-red-600 hover:bg-red-700 text-white px-4 py-2 rounded-lg"
            >
              <TrashIcon className="w-4 h-4" />
              Delete Pipeline
            </button>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-1 gap-6 mb-6">
          {/* Pipeline Steps */}
          <div className="bg-white rounded-lg shadow p-6">
            <button
              onClick={() => setStepsExpanded(!stepsExpanded)}
              className="w-full flex items-center gap-2 text-left hover:bg-gray-50 p-2 rounded -m-2 focus:outline-none"
            >
              <div className="flex-shrink-0">
                {stepsExpanded ? (
                  <ChevronUpIcon className="w-5 h-5 text-gray-400" />
                ) : (
                  <ChevronDownIcon className="w-5 h-5 text-gray-400" />
                )}
              </div>
              <CogIcon className="w-5 h-5 text-gray-700" />
              <h2 className="text-xl font-semibold text-gray-900">Pipeline Steps</h2>
              <span className="text-sm text-gray-500">({pipeline.steps.length} steps)</span>
            </button>
            
            {stepsExpanded && (
              <>
                {pipeline.steps.length === 0 ? (
                  <p className="text-gray-700">No steps defined.</p>
                ) : (
                  <div className="space-y-4">
                    {pipeline.steps.map((step, index) => (
                      <div key={index} className="border-l-4 border-blue-500 pl-4 p-3 bg-gray-50 rounded">
                        <h3 className="font-medium text-gray-900 mb-2">
                          {index + 1}. {step.name}
                        </h3>
                        
                        {step.description && (
                          <p className="text-sm text-gray-600 mb-2">{step.description}</p>
                        )}
                        
                        <div className="text-xs font-mono text-gray-900 bg-gray-100 p-2 rounded">
                          {step.command}
                        </div>
                        
                        <div className="text-xs text-gray-500 mt-2">
                          Timeout: {step.timeout || 300}s
                          {step.retry_count && ` • Retries: ${step.retry_count}`}
                          {step.continue_on_error && ` • Continue on error`}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </>
            )}
          </div>

          {/* Variables */}
          {pipeline.variables && Object.keys(pipeline.variables).length > 0 && (
            <div className="bg-white rounded-lg shadow p-6">
              <div className="flex items-center gap-2 mb-4">
                <CogIcon className="w-5 h-5 text-gray-700" />
                <h2 className="text-xl font-semibold text-gray-900">Variables</h2>
                <span className="text-sm text-gray-500">({Object.keys(pipeline.variables).length} variables)</span>
              </div>
              
              <div className="space-y-2">
                {Object.entries(pipeline.variables).map(([key, value]) => (
                  <div key={key} className="flex items-center justify-between p-2 bg-gray-50 rounded">
                    <span className="font-mono text-sm text-gray-700">${key}</span>
                    <span className="text-sm text-gray-900">{String(value)}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Pipeline Runs */}
        <div className="bg-white rounded-lg shadow p-6 mb-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-semibold text-gray-900">
              🏃 Pipeline Runs ({runs.length})
            </h2>
            <button
              onClick={loadData}
              className="flex items-center gap-2 text-blue-600 hover:text-blue-700"
            >
              <ArrowPathIcon className="w-4 h-4" />
              Refresh
            </button>
          </div>
          
          {runs.length === 0 ? (
            <div className="text-center py-8">
              <p className="text-gray-700 mb-4">No runs found for this pipeline.</p>
              <button
                onClick={runPipeline}
                className="flex items-center gap-2 bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-lg mx-auto"
              >
                <PlayIcon className="w-4 h-4" />
                Start First Run
              </button>
            </div>
          ) : (
            <>
              <div className="space-y-4 mb-4">
                {paginatedRuns.map((run) => (
                  <div key={run.id} className="border-l-4 border-green-500 pl-4 p-4 bg-gray-50 rounded">
                    <div className="flex items-center justify-between mb-2">
                      <button
                        onClick={() => router.push(`/run/${run.id}`)}
                        className="font-medium text-gray-900 hover:text-blue-600 text-left"
                      >
                        {run.name}
                      </button>
                      <StatusBadge status={run.status} />
                    </div>
                    
                    <div className="text-sm text-gray-600 space-y-1 mb-3">
                      <p><strong>Run ID:</strong> {run.id}</p>
                      <p><strong>Created:</strong> {formatDate(run.created_at)}</p>
                      {run.started_at && (
                        <p><strong>Started:</strong> {formatDate(run.started_at)}</p>
                      )}
                      {run.finished_at && (
                        <p><strong>Finished:</strong> {formatDate(run.finished_at)}</p>
                      )}
                      {run.total_duration && (
                        <p><strong>Duration:</strong> {formatDuration(run.total_duration)}</p>
                      )}
                    </div>
                    
                    <div className="flex flex-wrap gap-2">
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

        {/* Logs */}
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-semibold text-gray-900">📝 Activity Logs</h2>
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
