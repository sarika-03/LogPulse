import { Activity, Database, Cpu, Clock, Settings } from 'lucide-react';
import { useEffect, useState } from 'react';
import { BackendHealth } from '@/types/logs';
import { ConnectionStatus } from '@/components/ConnectionStatus';
import { ConnectionStatus as ConnectionStatusType } from '@/hooks/useBackendConnection';
import { ThemeToggle } from '@/components/ThemeToggle';

interface HeaderProps {
  health: BackendHealth | null;
  status: ConnectionStatusType;
  error: string | null;
  onSettingsClick: () => void;
  onReconnect: () => void;
}

export function Header({ health, status, error, onSettingsClick, onReconnect }: HeaderProps) {
  const [currentTime, setCurrentTime] = useState(new Date());
  const isConnected = status === 'connected';
  
  useEffect(() => {
    const timer = setInterval(() => setCurrentTime(new Date()), 1000);
    return () => clearInterval(timer);
  }, []);

  return (
    <header className="glass-panel border-b border-border px-6 py-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-3">
            <div className="relative">
              <Database className="h-8 w-8 text-primary" />
              <div className={`absolute -bottom-1 -right-1 h-3 w-3 rounded-full ${
                isConnected ? 'bg-success' : status === 'connecting' || status === 'reconnecting' ? 'bg-info animate-pulse' : 'bg-destructive'
              } ${isConnected ? 'pulse-dot' : ''}`} />
            </div>
            <div>
            <h1 className="text-xl font-bold text-foreground tracking-tight">
                LOG<span className="text-primary">PULSE</span>
              </h1>
              <p className="text-xs text-muted-foreground font-mono">
                {status === 'connected' ? 'Connected to backend' : 
                 status === 'connecting' ? 'Connecting...' :
                 status === 'reconnecting' ? 'Reconnecting...' :
                 status === 'error' ? 'Connection failed' :
                 'Backend not configured'}
              </p>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-6">
          {isConnected && health ? (
            <>
              <StatusIndicator 
                icon={<Activity className="h-4 w-4" />}
                label="Ingestion"
                value={`${health.ingestionRate}/s`}
                status="success"
              />
              <StatusIndicator 
                icon={<Database className="h-4 w-4" />}
                label="Chunks"
                value={health.chunksCount.toString()}
                status="success"
              />
              <StatusIndicator 
                icon={<Cpu className="h-4 w-4" />}
                label="Storage"
                value={formatBytes(health.storageUsed)}
                status={health.storageUsed > 1000000000 ? 'warning' : 'success'}
              />
            </>
          ) : (
            <ConnectionStatus
              status={status}
              health={health}
              error={error}
              onReconnect={onReconnect}
              onSettingsClick={onSettingsClick}
            />
          )}
          
          <div className="flex items-center gap-2 text-muted-foreground font-mono text-sm border-l border-border pl-6">
            <Clock className="h-4 w-4" />
            <span>{currentTime.toLocaleTimeString()}</span>
          </div>

          <div className="flex items-center gap-2">
            <ThemeToggle />
            
            <button
              onClick={onSettingsClick}
              className="p-2 rounded-lg hover:bg-muted transition-colors"
              aria-label="Settings"
            >
              <Settings className="h-5 w-5 text-muted-foreground hover:text-foreground" />
            </button>
          </div>
        </div>
      </div>
    </header>
  );
}

interface StatusIndicatorProps {
  icon: React.ReactNode;
  label: string;
  value: string;
  status: 'success' | 'warning' | 'error';
}

function StatusIndicator({ icon, label, value, status }: StatusIndicatorProps) {
  const statusColors = {
    success: 'text-success',
    warning: 'text-warning',
    error: 'text-destructive',
  };

  return (
    <div className="flex items-center gap-2">
      <span className={statusColors[status]}>{icon}</span>
      <div className="text-right">
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className={`text-sm font-mono font-semibold ${statusColors[status]}`}>{value}</p>
      </div>
    </div>
  );
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}
