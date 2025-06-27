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

import { PipelineRun, PipelineStep } from '@/types/api';
import { apiClient } from '@/lib/api';
import { formatDate, formatDuration, formatRelativeTime, copyToClipboard } from '@/lib/utils';
import { StatusBadge } from '@/components/ui/status-badge';
import { LogViewer, useLogs } from '@/components/ui/log-viewer';

interface RunDetailsPageProps {
  params: { id: string };
}

export default function RunDetailsPage({ params }: RunDetailsPageProps) {
  const [run, setRun] = useState<PipelineRun | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedSteps, setExpandedSteps] = useState<Set<number>>(new Set());
  const router = useRouter();
  const { logs, addLog, clearLogs } = useLogs();

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
      const runData = await apiClient.getRun(params.id);
      setRun(runData);
      setError(null);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load run data';
      setError(message);
      addLog(`❌ Error loading run data: ${message}`, 'error');
    } finally {
      setLoading(false);
    }
  };

  const cancelRun = async () => {
    if (!run) return;
    try {
      await apiClient.cancelRun(run.id);
      addLog(`🛑 Run cancelled: ${run.id}`, 'warning');
      loadData();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to cancel run';
      addLog(`❌ Failed to cancel run: ${message}`, 'error');
    }
  };

  const deleteRun = async () => {
    if (!run) return;
    if (!confirm(`Are you sure you want to delete run "${run.name}"? This action cannot be undone.`)) {
      return;
    }
    try {
      await apiClient.deleteRun(run.id);
      addLog(`🗑️ Run deleted: ${run.id}`, 'warning');
      router.push(`/pipeline/${run.pipeline_id}`);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to delete run';
      addLog(`❌ Failed to delete run: ${message}`, 'error');
    }
  };

  const toggleStep = (index: number) => {
    const newExpanded = new Set(expandedSteps);
    if (newExpanded.has(index)) {
      newExpanded.delete(index);
    } else {
      newExpanded.add(index);
    }
    setExpandedSteps(newExpanded);
  };

  const expandAllSteps = () => {
    if (!run?.steps) return;
    setExpandedSteps(new Set(Array.from({ length: run.steps.length }, (_, i) => i)));
  };

  const collapseAllSteps = () => {
    setExpandedSteps(new Set());
  };

  const rerunPipeline = async () => {
    if (!run) return;
    try {
      const result = await apiClient.runPipeline(run.pipeline_id);
      addLog(`🚀 Pipeline rerun started: ${run.pipeline_id}`, 'success');
      addLog(`🆕 New run created: ${result.run_id}`, 'info');
      // Navigate to the new run
      router.push(`/run/${result.run_id}`);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to rerun pipeline';
      addLog(`❌ Failed to rerun pipeline: ${message}`, 'error');
    }
  };

  const copyRunId = async () => {
    if (!run) return;
    try {
      await copyToClipboard(run.id);
      addLog(`📋 Run ID copied to clipboard: ${run.id}`, 'info');
    } catch (err) {
      addLog(`❌ Failed to copy run ID to clipboard`, 'error');
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <ArrowPathIcon className="w-8 h-8 animate-spin mx-auto mb-4 text-blue-600" />
          <p className="text-gray-600">Loading run...</p>
        </div>
      </div>
    );
  }

  if (error || !run) {
    return (
      <div className="min-h-screen bg-gray-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded mb-6">
            {error || 'Run not found'}
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
                Dashboard
              </Link>
              <ChevronRightIcon className="w-4 h-4 text-gray-400" />
              <Link
                href={`/pipeline/${run.pipeline_id}`}
                className="text-blue-600 hover:text-blue-700"
              >
                Pipeline
              </Link>
              <ChevronRightIcon className="w-4 h-4 text-gray-400" />
              <h1 className="text-2xl font-bold text-gray-900">{run.name}</h1>
            </div>
            <StatusBadge status={run.status} />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
            <div className="flex items-center gap-2 text-gray-600">
              <DocumentTextIcon className="w-4 h-4" />
              <button
                onClick={copyRunId}
                className="text-sm hover:text-blue-600 cursor-pointer text-left"
                title="Click to copy full run ID"
              >
                Run ID: {run.id.slice(0, 8)}... 📋
              </button>
            </div>
            <div className="flex items-center gap-2 text-gray-600">
              <CalendarIcon className="w-4 h-4" />
              <span className="text-sm">Created: {formatRelativeTime(run.created_at)}</span>
            </div>
            {run.started_at && (
              <div className="flex items-center gap-2 text-gray-600">
                <PlayIcon className="w-4 h-4" />
                <span className="text-sm">Started: {formatRelativeTime(run.started_at)}</span>
              </div>
            )}
            {run.total_duration && (
              <div className="flex items-center gap-2 text-gray-600">
                <ClockIcon className="w-4 h-4" />
                <span className="text-sm">Duration: {formatDuration(run.total_duration)}</span>
              </div>
            )}
          </div>

          {/* Action Buttons */}
          <div className="flex flex-wrap gap-3">
            <button
              onClick={rerunPipeline}
              className="flex items-center gap-2 bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-lg"
            >
              <PlayIcon className="w-4 h-4" />
              Rerun Pipeline
            </button>
            {run.status === 'running' && (
              <button
                onClick={cancelRun}
                className="flex items-center gap-2 bg-gray-600 hover:bg-gray-700 text-white px-4 py-2 rounded-lg"
              >
                <StopIcon className="w-4 h-4" />
                Cancel Run
              </button>
            )}
            <button
              onClick={() => loadData()}
              className="flex items-center gap-2 bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg"
            >
              <ArrowPathIcon className="w-4 h-4" />
              Refresh
            </button>
            <button
              onClick={deleteRun}
              className="flex items-center gap-2 bg-red-600 hover:bg-red-700 text-white px-4 py-2 rounded-lg"
            >
              <TrashIcon className="w-4 h-4" />
              Delete Run
            </button>
          </div>
        </div>

        {/* Run Steps */}
        <div className="bg-white rounded-lg shadow p-6 mb-6">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <CogIcon className="w-5 h-5 text-gray-700" />
              <h2 className="text-xl font-semibold text-gray-900">Run Steps</h2>
              <span className="text-sm text-gray-500">({run.steps?.length || 0} steps)</span>
            </div>
            
            <div className="flex gap-2">
              <button
                onClick={expandAllSteps}
                className="text-xs bg-gray-600 hover:bg-gray-700 text-white px-2 py-1 rounded"
              >
                Expand All
              </button>
              <button
                onClick={collapseAllSteps}
                className="text-xs bg-gray-600 hover:bg-gray-700 text-white px-2 py-1 rounded"
              >
                Collapse All
              </button>
            </div>
          </div>
          
          {!run.steps || run.steps.length === 0 ? (
            <p className="text-gray-700">No steps found for this run.</p>
          ) : (
            <div className="space-y-4">
              {run.steps.map((step, index) => {
                const isExpanded = expandedSteps.has(index);
                const stepDuration = step.start_time && step.end_time ? 
                  (new Date(step.end_time).getTime() - new Date(step.start_time).getTime()) / 1000 : null;
                
                return (
                  <div key={index} className="border border-gray-200 rounded-lg">
                    <button
                      onClick={() => toggleStep(index)}
                      className="w-full flex items-center justify-between p-4 text-left hover:bg-gray-50 focus:outline-none focus:bg-gray-50"
                    >
                      <div className="flex items-center gap-3">
                        <div className="flex-shrink-0">
                          {isExpanded ? (
                            <ChevronUpIcon className="w-5 h-5 text-gray-400" />
                          ) : (
                            <ChevronDownIcon className="w-5 h-5 text-gray-400" />
                          )}
                        </div>
                        <div>
                          <h3 className="font-medium text-gray-900">
                            {index + 1}. {step.name}
                          </h3>
                          {step.description && (
                            <p className="text-sm text-gray-600">{step.description}</p>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-3">
                        {stepDuration && (
                          <span className="text-xs text-gray-900">
                            {formatDuration(stepDuration)}
                          </span>
                        )}
                        {step.status && <StatusBadge status={step.status} />}
                      </div>
                    </button>
                    
                    {isExpanded && (
                      <div className="px-4 pb-4 border-t border-gray-200">
                        <div className="mt-4 space-y-4">
                          {/* Command */}
                          <div>
                            <h4 className="text-sm font-medium text-gray-900 mb-2">Command:</h4>
                            <div className="text-sm font-mono text-gray-900 bg-gray-100 p-3 rounded border border-gray-200">
                              {step.command}
                            </div>
                          </div>
                          
                          {/* Step Details */}
                          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                            <div className="text-gray-900">
                              <span className="font-medium text-gray-900">Timeout:</span> {step.timeout || 300}s
                            </div>
                            {step.retry_count !== undefined && (
                              <div className="text-gray-900">
                                <span className="font-medium text-gray-900">Retries:</span> {step.retry_count}
                              </div>
                            )}
                            {step.continue_on_error && (
                              <div className="text-gray-900">
                                <span className="font-medium text-gray-900">Continue on error:</span> Yes
                              </div>
                            )}
                          </div>
                          
                          {/* Timestamps */}
                          {(step.start_time || step.end_time) && (
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                              {step.start_time && (
                                <div className="text-gray-900">
                                  <span className="font-medium text-gray-900">Started:</span> {formatDate(step.start_time)}
                                </div>
                              )}
                              {step.end_time && (
                                <div className="text-gray-900">
                                  <span className="font-medium text-gray-900">Finished:</span> {formatDate(step.end_time)}
                                </div>
                              )}
                            </div>
                          )}
                          
                          {/* Output */}
                          {step.output && (
                            <div>
                              <h4 className="text-sm font-medium text-green-700 mb-2">Output:</h4>
                              <div className="bg-green-50 border border-green-200 p-3 rounded">
                                <pre className="text-sm font-mono text-green-800 whitespace-pre-wrap overflow-x-auto">
                                  {step.output}
                                </pre>
                              </div>
                            </div>
                          )}
                          
                          {/* Error */}
                          {step.error && (
                            <div>
                              <h4 className="text-sm font-medium text-red-700 mb-2">Error:</h4>
                              <div className="bg-red-50 border border-red-200 p-3 rounded">
                                <pre className="text-sm font-mono text-red-800 whitespace-pre-wrap overflow-x-auto">
                                  {step.error}
                                </pre>
                              </div>
                            </div>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
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
