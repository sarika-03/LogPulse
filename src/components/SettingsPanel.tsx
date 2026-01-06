import { useState, useEffect } from 'react';
import { X, Save, Trash2, CheckCircle, AlertCircle, Loader2, Wifi, ExternalLink, Copy, Server } from 'lucide-react';
import { BackendConfig } from '@/types/logs';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';

interface SettingsPanelProps {
  isOpen: boolean;
  onClose: () => void;
  config: BackendConfig | null;
  onSave: (config: BackendConfig) => void;
  onDisconnect: () => void;
}

export function SettingsPanel({ isOpen, onClose, config, onSave, onDisconnect }: SettingsPanelProps) {
  const [apiUrl, setApiUrl] = useState(config?.apiUrl || 'http://localhost:8080');
  const [apiKey, setApiKey] = useState(config?.apiKey || '');
  const [isTesting, setIsTesting] = useState(false);
  const [testResult, setTestResult] = useState<'success' | 'error' | null>(null);
  const [testMessage, setTestMessage] = useState<string>('');

  // Update form when config changes
  useEffect(() => {
    if (config) {
      setApiUrl(config.apiUrl);
      setApiKey(config.apiKey || '');
    }
  }, [config]);

  // Reset state when panel closes
  useEffect(() => {
    if (!isOpen) {
      setTestResult(null);
      setTestMessage('');
    }
  }, [isOpen]);

  if (!isOpen) return null;

  const validateUrl = (url: string): boolean => {
    try {
      const parsed = new URL(url);
      return parsed.protocol === 'http:' || parsed.protocol === 'https:';
    } catch {
      return false;
    }
  };

  const normalizeUrl = (url: string): string => {
    let normalized = url.trim();
    
    // Add protocol if missing
    if (!normalized.match(/^https?:\/\//)) {
      normalized = 'http://' + normalized;
    }
    
    // Remove trailing slash
    normalized = normalized.replace(/\/$/, '');
    
    return normalized;
  };

  const handleTest = async () => {
    const normalizedUrl = normalizeUrl(apiUrl);
    
    if (!validateUrl(normalizedUrl)) {
      setTestResult('error');
      setTestMessage('Invalid URL format. Must start with http:// or https://');
      toast.error('Invalid URL format');
      return;
    }

    setIsTesting(true);
    setTestResult(null);
    setTestMessage('');

    try {
      // Temporarily set config for testing
      const testConfig: BackendConfig = {
        apiUrl: normalizedUrl,
        apiKey: apiKey.trim() || undefined,
      };
      
      apiClient.setConfig(testConfig);
      
      console.log('[Settings] Testing connection to:', normalizedUrl);
      const health = await apiClient.health();
      
      setTestResult('success');
      setTestMessage(`Backend is ${health.status}. Uptime: ${Math.floor(health.uptime / 60)} minutes. Rate: ${health.ingestionRate}/s`);
      setApiUrl(normalizedUrl); // Update with normalized URL
      toast.success('Connection successful!');
      
      console.log('[Settings] Test successful:', health);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : 'Connection failed';
      setTestResult('error');
      setTestMessage(errorMsg);
      toast.error('Connection failed', { description: errorMsg });
      
      // Reset config on failure
      if (config) {
        apiClient.setConfig(config);
      } else {
        apiClient.clearConfig();
      }
      
      console.error('[Settings] Test failed:', errorMsg);
    } finally {
      setIsTesting(false);
    }
  };

  const handleSave = () => {
    const normalizedUrl = normalizeUrl(apiUrl);
    
    if (!validateUrl(normalizedUrl)) {
      toast.error('Invalid URL format', {
        description: 'Please enter a valid HTTP or HTTPS URL',
      });
      return;
    }

    const newConfig: BackendConfig = {
      apiUrl: normalizedUrl,
      apiKey: apiKey.trim() || undefined,
    };
    
    console.log('[Settings] Saving config:', newConfig);
    onSave(newConfig);
  };

  const handleDisconnect = () => {
    if (confirm('Are you sure you want to disconnect? All unsaved data will be lost.')) {
      onDisconnect();
      setApiUrl('http://localhost:8080');
      setApiKey('');
      setTestResult(null);
      setTestMessage('');
    }
  };

  const copyEndpoint = (endpoint: string) => {
    const fullUrl = `${apiUrl}${endpoint}`;
    navigator.clipboard.writeText(fullUrl);
    toast.success(`Copied: ${fullUrl}`);
  };

  const quickPresets = [
    { label: 'Local (8080)', url: 'http://localhost:8080' },
    { label: 'Local (8081)', url: 'http://localhost:8081' },
    { label: 'Docker', url: 'http://localhost:8082' },
  ];

  const endpoints = [
    { method: 'GET', path: '/health', description: 'Health check & system stats', color: 'text-success' },
    { method: 'GET', path: '/query', description: 'Query logs with filters', color: 'text-info' },
    { method: 'POST', path: '/ingest', description: 'Ingest new log entries', color: 'text-warning' },
    { method: 'GET', path: '/labels', description: 'List all label keys', color: 'text-primary' },
    { method: 'GET', path: '/metrics', description: 'Prometheus-format metrics', color: 'text-primary' },
    { method: 'WS', path: '/stream', description: 'Real-time log streaming', color: 'text-purple-400' },
  ];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-background/80 backdrop-blur-sm" onClick={onClose} />
      
      <div className="relative glass-panel rounded-xl w-full max-w-2xl p-6 shadow-2xl border border-border animate-fade-in max-h-[90vh] overflow-auto scrollbar-thin">
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-primary/10">
              <Server className="h-6 w-6 text-primary" />
            </div>
            <div>
              <h2 className="text-xl font-bold">Backend Configuration</h2>
              <p className="text-sm text-muted-foreground">Connect to your LogPulse Go backend</p>
            </div>
          </div>
          <button 
            onClick={onClose} 
            className="p-2 hover:bg-muted rounded-lg transition-colors"
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="space-y-6">
          {/* Quick Presets */}
          <div>
            <label className="block text-sm font-medium mb-2">Quick Presets</label>
            <div className="flex gap-2">
              {quickPresets.map((preset) => (
                <button
                  key={preset.url}
                  onClick={() => setApiUrl(preset.url)}
                  className={`px-3 py-1.5 rounded-lg text-xs font-mono transition-colors ${
                    apiUrl === preset.url
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-muted text-muted-foreground hover:text-foreground hover:bg-muted/80'
                  }`}
                >
                  {preset.label}
                </button>
              ))}
            </div>
          </div>

          {/* Connection Form */}
          <div className="grid gap-4">
            <div>
              <label className="block text-sm font-medium mb-2">
                Backend API URL <span className="text-destructive">*</span>
              </label>
              <input
                type="text"
                value={apiUrl}
                onChange={(e) => setApiUrl(e.target.value)}
                onBlur={() => setApiUrl(normalizeUrl(apiUrl))}
                placeholder="http://localhost:8080"
                className="query-input w-full"
                autoComplete="off"
              />
              <p className="text-xs text-muted-foreground mt-1">
                The URL where your Go backend is running (protocol required)
              </p>
            </div>

            <div>
              <label className="block text-sm font-medium mb-2">
                API Key <span className="text-muted-foreground">(optional)</span>
              </label>
              <input
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder="Enter API key if required"
                className="query-input w-full"
                autoComplete="off"
              />
              <p className="text-xs text-muted-foreground mt-1">
                Leave empty if your backend doesn't require authentication
              </p>
            </div>
          </div>

          {/* Test Result */}
          {testResult && (
            <div 
              className={`p-3 rounded-lg border animate-fade-in ${
                testResult === 'success' 
                  ? 'bg-success/10 border-success/30' 
                  : 'bg-destructive/10 border-destructive/30'
              }`}
            >
              <div className="flex items-center gap-2">
                {testResult === 'success' ? (
                  <CheckCircle className="h-4 w-4 text-success" />
                ) : (
                  <AlertCircle className="h-4 w-4 text-destructive" />
                )}
                <span className={`text-sm font-medium ${
                  testResult === 'success' ? 'text-success' : 'text-destructive'
                }`}>
                  {testResult === 'success' ? 'Connection Successful ✓' : 'Connection Failed ✗'}
                </span>
              </div>
              {testMessage && (
                <p className={`text-xs mt-1 ${
                  testResult === 'success' ? 'text-success/80' : 'text-destructive/80'
                }`}>
                  {testMessage}
                </p>
              )}
            </div>
          )}

          {/* Action Buttons */}
          <div className="flex items-center gap-3">
            <Button
              onClick={handleTest}
              disabled={isTesting || !apiUrl}
              variant="secondary"
              className="flex items-center gap-2"
            >
              {isTesting ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : testResult === 'success' ? (
                <CheckCircle className="h-4 w-4 text-success" />
              ) : testResult === 'error' ? (
                <AlertCircle className="h-4 w-4 text-destructive" />
              ) : (
                <Wifi className="h-4 w-4" />
              )}
              {isTesting ? 'Testing...' : 'Test Connection'}
            </Button>

            <Button
              onClick={handleSave}
              disabled={!apiUrl || isTesting}
              className="flex items-center gap-2"
            >
              <Save className="h-4 w-4" />
              Save & Connect
            </Button>

            {config && (
              <Button
                onClick={handleDisconnect}
                variant="destructive"
                className="ml-auto flex items-center gap-2"
              >
                <Trash2 className="h-4 w-4" />
                Disconnect
              </Button>
            )}
          </div>

          {/* Endpoints Reference */}
          <div className="border-t border-border pt-6">
            <h3 className="text-sm font-semibold mb-3 flex items-center gap-2">
              <ExternalLink className="h-4 w-4 text-primary" />
              Expected API Endpoints
            </h3>
            <div className="grid gap-2">
              {endpoints.map((ep) => (
                <div 
                  key={ep.path}
                  className="flex items-center justify-between p-2 rounded-lg bg-muted/50 hover:bg-muted transition-colors group"
                >
                  <div className="flex items-center gap-3 flex-1 min-w-0">
                    <span className={`text-xs font-mono font-bold ${ep.color} w-12 flex-shrink-0`}>
                      {ep.method}
                    </span>
                    <code className="text-sm font-mono truncate">{ep.path}</code>
                    <span className="text-xs text-muted-foreground hidden lg:inline truncate">
                      — {ep.description}
                    </span>
                  </div>
                  <button
                    onClick={() => copyEndpoint(ep.path)}
                    className="p-1 opacity-0 group-hover:opacity-100 transition-opacity hover:bg-background rounded flex-shrink-0"
                    title="Copy full URL"
                  >
                    <Copy className="h-3.5 w-3.5 text-muted-foreground" />
                  </button>
                </div>
              ))}
            </div>
          </div>

          {/* Quick Start Guide */}
          <div className="border-t border-border pt-6">
            <h3 className="text-sm font-semibold mb-3">🚀 Quick Start</h3>
            <div className="text-xs text-muted-foreground space-y-2 font-mono bg-muted/50 p-4 rounded-lg">
              <p className="text-foreground font-semibold"># Start the Go backend:</p>
              <p className="text-primary">cd backend && go run cmd/server/main.go</p>
              
              <p className="text-foreground font-semibold mt-3"># Or with Docker:</p>
              <p className="text-primary">cd backend/docker && docker-compose up -d</p>
              
              <p className="text-foreground font-semibold mt-3"># Then connect here:</p>
              <p className="text-success">http://localhost:8080</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}


