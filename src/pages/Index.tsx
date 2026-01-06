import { useState, useCallback, useEffect } from 'react';
import { Header } from '@/components/Header';
import { QueryBar } from '@/components/QueryBar';
import { LogViewer } from '@/components/LogViewer';
import { TestPanel } from '@/components/TestPanel';
import { SettingsPanel } from '@/components/SettingsPanel';
import { LabelsExplorer } from '@/components/LabelsExplorer';
import { MetricsDashboard } from '@/components/MetricsDashboard';
import { AlertConfig } from '@/components/AlertConfig';
import { LiveStream } from '@/components/LiveStream';
import { AnalyticsCharts } from '@/components/AnalyticsCharts';
import { SavedSearches } from '@/components/SavedSearches';
import { useBackendConnection } from '@/hooks/useBackendConnection';
import { LogEntry, BackendConfig, QueryResult } from '@/types/logs';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';
import { Search, Tag, Activity, Bell, FlaskConical, Radio, BarChart3, Zap } from 'lucide-react';

type TabType = 'logs' | 'live' | 'labels' | 'metrics' | 'analytics' | 'alerts';
type RightPanelType = 'test' | null;

const Index = () => {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [queryStats, setQueryStats] = useState<QueryResult['stats'] | undefined>();
  const [isLoading, setIsLoading] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [activeTab, setActiveTab] = useState<TabType>('logs');
  const [rightPanel, setRightPanel] = useState<RightPanelType>('test');
  const [currentQuery, setCurrentQuery] = useState('{service="api-gateway"}');
  const [currentTimeRange, setCurrentTimeRange] = useState('1h');

  // Use the backend connection hook
  const { 
    status, 
    health, 
    config, 
    error, 
    connect, 
    disconnect, 
    reconnect, 
    isConnected 
  } = useBackendConnection();

  // Auto-open settings if not connected on mount
  useEffect(() => {
    const hasConfig = localStorage.getItem('logpulse_config');
    if (!hasConfig && !isConnected) {
      // Show settings after a short delay to allow initial render
      const timer = setTimeout(() => {
        setShowSettings(true);
        toast.info('Welcome to LogPulse! Configure your backend to get started.');
      }, 500);
      return () => clearTimeout(timer);
    }
  }, [isConnected]);

  // Auto-refresh logs when tab becomes visible again
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (!document.hidden && isConnected && activeTab === 'logs' && logs.length > 0) {
        handleRefresh();
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange);
  }, [isConnected, activeTab, logs.length]);

  const parseQuery = (query: string): Record<string, string> => {
    const labels: Record<string, string> = {};
    
    // Handle empty query
    if (!query || query === '{}') {
      return labels;
    }

    try {
      const match = query.match(/\{(.+)\}/);
      if (match) {
        const pairs = match[1].split(',');
        pairs.forEach((pair) => {
          const [key, value] = pair.split('=').map((s) => s.trim().replace(/["']/g, ''));
          if (key && value) {
            labels[key] = value;
          }
        });
      }
    } catch (err) {
      console.error('[Query] Parse error:', err);
      toast.error('Invalid query syntax');
    }

    return labels;
  };

  const getTimeRange = (range: string): { start: string; end: string } => {
    const now = new Date();
    const durations: Record<string, number> = {
      '5m': 5 * 60 * 1000,
      '15m': 15 * 60 * 1000,
      '1h': 60 * 60 * 1000,
      '6h': 6 * 60 * 60 * 1000,
      '24h': 24 * 60 * 60 * 1000,
    };

    const duration = durations[range] || durations['1h'];
    const start = new Date(now.getTime() - duration);

    return {
      start: start.toISOString(),
      end: now.toISOString(),
    };
  };

  const handleQuery = useCallback(async (query: string, timeRange: string) => {
    if (!isConnected) {
      toast.error('Not connected to backend', {
        description: 'Click the settings icon to configure your connection',
      });
      return;
    }

    setCurrentQuery(query);
    setCurrentTimeRange(timeRange);
    setIsLoading(true);
    
    try {
      const labels = parseQuery(query);
      const { start, end } = getTimeRange(timeRange);
      
      console.log('[Query] Executing:', { query, labels, start, end });
      const result = await apiClient.query(labels, start, end, 1000);
      
      setLogs(result.logs);
      setQueryStats(result.stats);
      
      console.log('[Query] Results:', { count: result.logs.length, stats: result.stats });
      
      if (result.logs.length === 0) {
        toast.info('No logs found', {
          description: 'Try adjusting your query or time range',
        });
      } else {
        toast.success(`Found ${result.logs.length} logs`, {
          description: `Scanned ${result.stats.scannedLines} lines in ${result.stats.executionTime}ms`,
        });
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Query failed';
      console.error('[Query] Error:', errorMessage);
      toast.error('Query failed', { description: errorMessage });
      setLogs([]);
      setQueryStats(undefined);
    } finally {
      setIsLoading(false);
    }
  }, [isConnected]);

  const handleRefresh = useCallback(() => {
    if (isConnected) {
      handleQuery(currentQuery, currentTimeRange);
    } else {
      toast.error('Cannot refresh: Not connected to backend');
    }
  }, [isConnected, handleQuery, currentQuery, currentTimeRange]);

  const handleSaveConfig = async (newConfig: BackendConfig) => {
    const success = await connect(newConfig);
    if (success) {
      setShowSettings(false);
      // Auto-run initial query after successful connection
      setTimeout(() => {
        handleQuery(currentQuery, currentTimeRange);
      }, 500);
    }
  };

  const handleDisconnect = () => {
    disconnect();
    setLogs([]);
    setQueryStats(undefined);
    setShowSettings(false);
    toast.info('Disconnected from backend');
  };

  const tabs: { id: TabType; label: string; icon: React.ReactNode; badge?: React.ReactNode }[] = [
    { id: 'logs', label: 'Logs', icon: <Search className="h-4 w-4" /> },
    { 
      id: 'live', 
      label: 'Live', 
      icon: <Radio className="h-4 w-4" />,
      badge: isConnected && activeTab === 'live' ? (
        <span className="h-2 w-2 rounded-full bg-success animate-pulse" />
      ) : null
    },
    { id: 'labels', label: 'Labels', icon: <Tag className="h-4 w-4" /> },
    { id: 'metrics', label: 'Metrics', icon: <Activity className="h-4 w-4" /> },
    { id: 'analytics', label: 'Analytics', icon: <BarChart3 className="h-4 w-4" /> },
    { id: 'alerts', label: 'Alerts', icon: <Bell className="h-4 w-4" /> },
  ];

  return (
    <div className="h-screen flex flex-col bg-background overflow-hidden">
      <Header 
        health={health}
        status={status}
        error={error}
        onSettingsClick={() => setShowSettings(true)}
        onReconnect={reconnect}
      />
      
      <div className="flex flex-1 overflow-hidden">
        <main className="flex-1 flex flex-col overflow-hidden">
          {/* Tab navigation */}
          <div className="flex items-center border-b border-border px-4 bg-card/30">
            <div className="flex">
              {tabs.map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  disabled={!isConnected && tab.id !== 'logs'}
                  className={`flex items-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-all ${
                    activeTab === tab.id
                      ? 'border-primary text-primary'
                      : 'border-transparent text-muted-foreground hover:text-foreground'
                  } ${!isConnected && tab.id !== 'logs' ? 'opacity-50 cursor-not-allowed' : ''}`}
                >
                  {tab.icon}
                  {tab.label}
                  {tab.badge}
                </button>
              ))}
            </div>
            
            <div className="ml-auto flex items-center gap-2">
              {!isConnected && (
                <button
                  onClick={() => setShowSettings(true)}
                  className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-primary/10 text-primary text-sm font-medium hover:bg-primary/20 transition-colors animate-pulse"
                >
                  <Zap className="h-4 w-4" />
                  Connect Backend
                </button>
              )}
              
              <button
                onClick={() => setRightPanel(rightPanel === 'test' ? null : 'test')}
                disabled={!isConnected}
                className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                  rightPanel === 'test'
                    ? 'bg-primary/10 text-primary'
                    : 'text-muted-foreground hover:text-foreground hover:bg-muted'
                } ${!isConnected ? 'opacity-50 cursor-not-allowed' : ''}`}
              >
                <FlaskConical className="h-4 w-4" />
                Test Panel
              </button>
            </div>
          </div>
          
          {/* Tab content */}
          <div className="flex-1 overflow-hidden flex">
            <div className="flex-1 flex flex-col overflow-hidden">
              {activeTab === 'logs' && (
                <>
                  <QueryBar
                    onQuery={handleQuery}
                    onRefresh={handleRefresh}
                    isLoading={isLoading}
                    isConnected={isConnected}
                  />
                  
                  <div className="flex-1 overflow-hidden flex">
                    {/* Saved Searches Sidebar */}
                    {isConnected && (
                      <div className="w-64 border-r border-border bg-card/50">
                        <SavedSearches
                          onExecuteSearch={handleQuery}
                          currentQuery={currentQuery}
                          currentTimeRange={currentTimeRange}
                        />
                      </div>
                    )}
                    
                    <div className="flex-1 flex flex-col overflow-hidden">
                      {isConnected && (
                        <div className="px-6 py-2 border-b border-border flex items-center justify-between bg-muted/30">
                          <span className="text-sm text-muted-foreground font-mono">
                            {logs.length > 0 ? `Showing ${logs.length} logs` : 'No logs yet'}
                          </span>
                          {health && (
                            <div className="flex items-center gap-4 text-xs text-muted-foreground font-mono">
                              <span>Backend: {health.status}</span>
                              <span>•</span>
                              <span>Rate: {health.ingestionRate}/s</span>
                              <span>•</span>
                              <span>Chunks: {health.chunksCount}</span>
                            </div>
                          )}
                        </div>
                      )}
                      
                      <LogViewer 
                        logs={logs} 
                        isLoading={isLoading}
                        isConnected={isConnected}
                        queryStats={queryStats}
                      />
                    </div>
                  </div>
                </>
              )}

              {activeTab === 'live' && (
                <LiveStream isConnected={isConnected} />
              )}

              {activeTab === 'labels' && (
                <LabelsExplorer isConnected={isConnected} />
              )}

              {activeTab === 'metrics' && (
                <MetricsDashboard isConnected={isConnected} />
              )}

              {activeTab === 'analytics' && (
                <AnalyticsCharts isConnected={isConnected} />
              )}

              {activeTab === 'alerts' && (
                <AlertConfig isConnected={isConnected} />
              )}
            </div>

            {/* Right panel */}
            {rightPanel === 'test' && isConnected && (
              <TestPanel isConnected={isConnected} />
            )}
          </div>
        </main>
      </div>

      <SettingsPanel
        isOpen={showSettings}
        onClose={() => setShowSettings(false)}
        config={config}
        onSave={handleSaveConfig}
        onDisconnect={handleDisconnect}
      />
    </div>
  );
};

export default Index;



