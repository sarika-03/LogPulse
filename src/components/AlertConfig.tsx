import { useState, useEffect } from 'react';
import { 
  Bell, 
  Plus, 
  Trash2, 
  AlertTriangle, 
  AlertCircle, 
  Info, 
  Power, 
  PowerOff,
  Save,
  X,
  Loader2
} from 'lucide-react';
import { apiClient, AlertRule } from '@/lib/api';
import { toast } from 'sonner';

interface AlertConfigProps {
  isConnected: boolean;
}

export function AlertConfig({ isConnected }: AlertConfigProps) {
  const [alerts, setAlerts] = useState<AlertRule[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    query: '{level="error"}',
    condition: 'gt' as const,
    threshold: 10,
    duration: '5m',
    severity: 'warning' as const,
    webhook: '',
  });

  const fetchAlerts = async () => {
    if (!isConnected) return;
    
    setIsLoading(true);
    try {
      const data = await apiClient.getAlerts();
      setAlerts(data);
    } catch (error) {
      // Alerts endpoint might not exist yet - that's okay
      console.log('Alerts endpoint not available');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    if (isConnected) {
      fetchAlerts();
    }
  }, [isConnected]);

  const handleCreate = async () => {
    if (!formData.name || !formData.query) {
      toast.error('Name and query are required');
      return;
    }

    try {
      const newAlert = await apiClient.createAlert({
        ...formData,
        enabled: true,
        webhook: formData.webhook || undefined,
      });
      setAlerts([...alerts, newAlert]);
      setShowForm(false);
      resetForm();
      toast.success('Alert rule created');
    } catch (error) {
      toast.error('Failed to create alert. Ensure backend supports /alerts endpoint.');
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await apiClient.deleteAlert(id);
      setAlerts(alerts.filter((a) => a.id !== id));
      toast.success('Alert deleted');
    } catch (error) {
      toast.error('Failed to delete alert');
    }
  };

  const handleToggle = async (id: string, enabled: boolean) => {
    try {
      await apiClient.updateAlertStatus(id, enabled);
      setAlerts(alerts.map((a) => (a.id === id ? { ...a, enabled } : a)));
      toast.success(`Alert ${enabled ? 'enabled' : 'disabled'}`);
    } catch (error) {
      toast.error('Failed to update alert');
    }
  };

  const resetForm = () => {
    setFormData({
      name: '',
      query: '{level="error"}',
      condition: 'gt',
      threshold: 10,
      duration: '5m',
      severity: 'warning',
      webhook: '',
    });
  };

  const severityIcons = {
    critical: <AlertCircle className="h-4 w-4 text-destructive" />,
    warning: <AlertTriangle className="h-4 w-4 text-warning" />,
    info: <Info className="h-4 w-4 text-info" />,
  };

  const severityColors = {
    critical: 'bg-destructive/10 border-destructive/30 text-destructive',
    warning: 'bg-warning/10 border-warning/30 text-warning',
    info: 'bg-info/10 border-info/30 text-info',
  };

  const conditionLabels: Record<string, string> = {
    gt: '>',
    lt: '<',
    eq: '=',
    gte: '≥',
    lte: '≤',
  };

  if (!isConnected) {
    return (
      <div className="p-6 text-center text-muted-foreground">
        <p className="text-sm">Connect to backend to configure alerts</p>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col">
      <div className="p-4 border-b border-border flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
            Alert Rules
          </h3>
          <p className="text-xs text-muted-foreground mt-1">
            Configure log-based alerting
          </p>
        </div>
        <button
          onClick={() => setShowForm(true)}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:opacity-90"
        >
          <Plus className="h-4 w-4" />
          New Rule
        </button>
      </div>

      {/* Alert creation form */}
      {showForm && (
        <div className="p-4 border-b border-border bg-muted/30 space-y-3">
          <div className="flex items-center justify-between">
            <h4 className="text-sm font-medium">Create Alert Rule</h4>
            <button onClick={() => { setShowForm(false); resetForm(); }} className="p-1 hover:bg-muted rounded">
              <X className="h-4 w-4" />
            </button>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium mb-1">Name</label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                placeholder="High error rate"
                className="query-input w-full text-sm py-2 text-foreground"
              />
            </div>
            <div>
              <label className="block text-xs font-medium mb-1">Severity</label>
              <select
                value={formData.severity}
                onChange={(e) => setFormData({ ...formData, severity: e.target.value as any })}
                className="query-input w-full text-sm py-2 text-foreground"
              >
                <option value="critical">Critical</option>
                <option value="warning">Warning</option>
                <option value="info">Info</option>
              </select>
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium mb-1">Query</label>
            <input
              type="text"
              value={formData.query}
              onChange={(e) => setFormData({ ...formData, query: e.target.value })}
              placeholder='{level="error", service="api"}'
              className="query-input w-full text-sm py-2 font-mono text-foreground"
            />
          </div>

          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="block text-xs font-medium mb-1">Condition</label>
              <select
                value={formData.condition}
                onChange={(e) => setFormData({ ...formData, condition: e.target.value as any })}
                className="query-input w-full text-sm py-2 text-foreground"
              >
                <option value="gt">Greater than (&gt;)</option>
                <option value="gte">Greater or equal (≥)</option>
                <option value="lt">Less than (&lt;)</option>
                <option value="lte">Less or equal (≤)</option>
                <option value="eq">Equal (=)</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium mb-1">Threshold</label>
              <input
                type="number"
                value={formData.threshold}
                onChange={(e) => setFormData({ ...formData, threshold: parseInt(e.target.value) })}
                className="query-input w-full text-sm py-2 text-foreground"
              />
            </div>
            <div>
              <label className="block text-xs font-medium mb-1">Duration</label>
              <select
                value={formData.duration}
                onChange={(e) => setFormData({ ...formData, duration: e.target.value })}
                className="query-input w-full text-sm py-2 text-foreground"
              >
                <option value="1m">1 minute</option>
                <option value="5m">5 minutes</option>
                <option value="15m">15 minutes</option>
                <option value="1h">1 hour</option>
              </select>
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium mb-1">Webhook URL (optional)</label>
            <input
              type="text"
              value={formData.webhook}
              onChange={(e) => setFormData({ ...formData, webhook: e.target.value })}
              placeholder="https://hooks.slack.com/..."
              className="query-input w-full text-sm py-2 text-foreground"
            />
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <button
              onClick={() => { setShowForm(false); resetForm(); }}
              className="px-4 py-2 rounded-lg bg-muted text-muted-foreground text-sm font-medium hover:bg-muted/80"
            >
              Cancel
            </button>
            <button
              onClick={handleCreate}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:opacity-90"
            >
              <Save className="h-4 w-4" />
              Create Rule
            </button>
          </div>
        </div>
      )}

      {/* Alerts list */}
      <div className="flex-1 overflow-auto scrollbar-thin">
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        ) : alerts.length === 0 ? (
          <div className="p-6 text-center">
            <Bell className="h-8 w-8 mx-auto mb-2 text-muted-foreground opacity-50" />
            <p className="text-sm text-muted-foreground">No alert rules configured</p>
            <p className="text-xs text-muted-foreground mt-1">
              Create rules to get notified when log patterns match
            </p>
          </div>
        ) : (
          <div className="divide-y divide-border/50">
            {alerts.map((alert) => (
              <div
                key={alert.id}
                className={`p-4 transition-colors ${!alert.enabled ? 'opacity-50' : ''}`}
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-start gap-3 flex-1">
                    {severityIcons[alert.severity]}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <h4 className="font-medium text-sm">{alert.name}</h4>
                        <span className={`text-xs px-2 py-0.5 rounded border ${severityColors[alert.severity]}`}>
                          {alert.severity}
                        </span>
                      </div>
                      <p className="font-mono text-xs text-muted-foreground mt-1 truncate">
                        {alert.query}
                      </p>
                      <p className="text-xs text-muted-foreground mt-1">
                        Alert when count {conditionLabels[alert.condition]} {alert.threshold} in {alert.duration}
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handleToggle(alert.id, !alert.enabled)}
                      className={`p-1.5 rounded transition-colors ${
                        alert.enabled
                          ? 'text-success hover:bg-success/10'
                          : 'text-muted-foreground hover:bg-muted'
                      }`}
                    >
                      {alert.enabled ? <Power className="h-4 w-4" /> : <PowerOff className="h-4 w-4" />}
                    </button>
                    <button
                      onClick={() => handleDelete(alert.id)}
                      className="p-1.5 rounded text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Info footer */}
      <div className="p-3 border-t border-border bg-muted/30">
        <p className="text-xs text-muted-foreground">
          <strong>Note:</strong> Alerts require /alerts endpoint in your Go backend.
          Webhooks support Slack, Discord, PagerDuty formats.
        </p>
      </div>
    </div>
  );
}
