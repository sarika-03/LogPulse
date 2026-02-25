import { BackendConfig, BackendHealth, LogEntry, IngestPayload, QueryResult } from '@/types/logs';

class ApiClient {
  private config: BackendConfig | null = null;

  setConfig(config: BackendConfig) {
    this.config = config;
    localStorage.setItem('logpulse_config', JSON.stringify(config));
  }

  getConfig(): BackendConfig | null {
    if (this.config) return this.config;
    const stored = localStorage.getItem('logpulse_config');
    if (stored) {
      this.config = JSON.parse(stored);
      return this.config;
    }
    return null;
  }

  clearConfig() {
    this.config = null;
    localStorage.removeItem('logpulse_config');
  }

  private getHeaders(): HeadersInit {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    };
    if (this.config?.apiKey) {
      headers['X-API-Key'] = this.config.apiKey;
    }
    return headers;
  }

  private getBaseUrl(): string {
    if (!this.config?.apiUrl) {
      throw new Error('Backend not configured');
    }
    return this.config.apiUrl.replace(/\/$/, '');
  }

  async health(): Promise<BackendHealth> {
    const response = await fetch(`${this.getBaseUrl()}/health`, {
      headers: this.getHeaders(),
    });
    if (!response.ok) {
      throw new Error(`Health check failed: ${response.status}`);
    }
    return response.json();
  }

  async query(
    labels: Record<string, string>,
    start?: string,
    end?: string,
    limit: number = 100,
    offset: number = 0
  ): Promise<QueryResult> {
    const params = new URLSearchParams();
    
    const labelParts = Object.entries(labels)
      .map(([k, v]) => `${k}="${v}"`)
      .join(', ');
    params.set('query', `{${labelParts}}`);
    
    if (start) params.set('start', start);
    if (end) params.set('end', end);
    params.set('limit', limit.toString());
    params.set('offset', offset.toString());

    const response = await fetch(`${this.getBaseUrl()}/query?${params}`, {
      headers: this.getHeaders(),
    });
    
    if (!response.ok) {
      throw new Error(`Query failed: ${response.status}`);
    }
    return response.json();
  }

  async ingest(payload: IngestPayload): Promise<{ accepted: number }> {
    const response = await fetch(`${this.getBaseUrl()}/ingest`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify(payload),
    });
    
    if (!response.ok) {
      throw new Error(`Ingest failed: ${response.status}`);
    }
    return response.json();
  }

  async getLabels(): Promise<string[]> {
    const response = await fetch(`${this.getBaseUrl()}/labels`, {
      headers: this.getHeaders(),
    });
    if (!response.ok) {
      throw new Error(`Failed to get labels: ${response.status}`);
    }
    return response.json();
  }

  async getLabelValues(label: string): Promise<string[]> {
    const response = await fetch(`${this.getBaseUrl()}/labels/${label}/values`, {
      headers: this.getHeaders(),
    });
    if (!response.ok) {
      throw new Error(`Failed to get label values: ${response.status}`);
    }
    return response.json();
  }

  async getMetrics(): Promise<string> {
    const response = await fetch(`${this.getBaseUrl()}/metrics`, {
      headers: this.getHeaders(),
    });
    if (!response.ok) {
      throw new Error(`Failed to get metrics: ${response.status}`);
    }
    return response.text();
  }

  async getAlerts(): Promise<AlertRule[]> {
    const response = await fetch(`${this.getBaseUrl()}/alerts`, {
      headers: this.getHeaders(),
    });
    if (!response.ok) {
      throw new Error(`Failed to get alerts: ${response.status}`);
    }
    return response.json();
  }

  async createAlert(alert: Omit<AlertRule, 'id' | 'createdAt'>): Promise<AlertRule> {
    const response = await fetch(`${this.getBaseUrl()}/alerts`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify(alert),
    });
    if (!response.ok) {
      throw new Error(`Failed to create alert: ${response.status}`);
    }
    return response.json();
  }

  async deleteAlert(id: string): Promise<void> {
    const response = await fetch(`${this.getBaseUrl()}/alerts/${id}`, {
      method: 'DELETE',
      headers: this.getHeaders(),
    });
    if (!response.ok) {
      throw new Error(`Failed to delete alert: ${response.status}`);
    }
  }

  async updateAlertStatus(id: string, enabled: boolean): Promise<void> {
    const response = await fetch(`${this.getBaseUrl()}/alerts/${id}/status`, {
      method: 'PATCH',
      headers: this.getHeaders(),
      body: JSON.stringify({ enabled }),
    });
    if (!response.ok) {
      throw new Error(`Failed to update alert: ${response.status}`);
    }
  }
}

export interface AlertRule {
  id: string;
  name: string;
  query: string;
  condition: 'gt' | 'lt' | 'eq' | 'gte' | 'lte';
  threshold: number;
  duration: string;
  severity: 'critical' | 'warning' | 'info';
  enabled: boolean;
  createdAt: string;
  webhook?: string;
}

export const apiClient = new ApiClient();
