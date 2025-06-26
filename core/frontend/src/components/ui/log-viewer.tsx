'use client';

import { useEffect, useRef, useState } from 'react';
import { cn } from '@/lib/utils';

interface LogEntry {
  timestamp: string;
  message: string;
  level?: 'info' | 'error' | 'warning' | 'success';
}

interface LogViewerProps {
  logs: LogEntry[];
  className?: string;
  height?: string;
  autoScroll?: boolean;
}

export function LogViewer({ 
  logs, 
  className, 
  height = 'h-80',
  autoScroll = true 
}: LogViewerProps) {
  const logEndRef = useRef<HTMLDivElement>(null);
  const [isUserScrolling, setIsUserScrolling] = useState(false);

  useEffect(() => {
    if (autoScroll && !isUserScrolling) {
      logEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, autoScroll, isUserScrolling]);

  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    const { scrollTop, scrollHeight, clientHeight } = e.currentTarget;
    const isAtBottom = scrollHeight - scrollTop === clientHeight;
    setIsUserScrolling(!isAtBottom);
  };

  const getLevelColor = (level?: string) => {
    switch (level) {
      case 'error':
        return 'text-red-400';
      case 'warning':
        return 'text-yellow-400';
      case 'success':
        return 'text-green-400';
      default:
        return 'text-green-300';
    }
  };

  return (
    <div 
      className={cn(
        'bg-black rounded-lg border border-gray-700 overflow-hidden',
        className
      )}
    >
      <div 
        className={cn(
          'p-4 overflow-y-auto font-mono text-sm',
          height
        )}
        onScroll={handleScroll}
      >
        {logs.length === 0 ? (
          <div className="text-gray-500">No logs available...</div>
        ) : (
          logs.map((log, index) => (
            <div 
              key={index} 
              className={cn(
                'whitespace-pre-wrap break-words',
                getLevelColor(log.level)
              )}
            >
              <span className="text-gray-400 text-xs">
                [{log.timestamp}]
              </span>{' '}
              {log.message}
            </div>
          ))
        )}
        <div ref={logEndRef} />
      </div>
      
      {isUserScrolling && (
        <div className="absolute bottom-2 right-2">
          <button
            onClick={() => {
              logEndRef.current?.scrollIntoView({ behavior: 'smooth' });
              setIsUserScrolling(false);
            }}
            className="bg-blue-600 hover:bg-blue-700 text-white text-xs px-2 py-1 rounded"
          >
            ↓ Scroll to bottom
          </button>
        </div>
      )}
    </div>
  );
}

// Hook for managing logs
export function useLogs() {
  const [logs, setLogs] = useState<LogEntry[]>([]);

  const addLog = (message: string, level?: LogEntry['level']) => {
    const timestamp = new Date().toISOString();
    setLogs(prev => {
      const newLogs = [...prev, { timestamp, message, level }];
      // Keep only last 100 logs to prevent memory issues
      return newLogs.slice(-100);
    });
  };

  const clearLogs = () => {
    setLogs([]);
  };

  return { logs, addLog, clearLogs };
} 
