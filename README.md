# LogPulse Frontend

Modern React-based web interface for LogPulse log aggregation system.

## 🚀 Features

- **Real-time Log Streaming** - WebSocket-based live log tailing
- **Advanced Query Interface** - LogQL-inspired query language
- **Analytics Dashboard** - Visualize log patterns and trends
- **Anomaly Detection** - Automatic detection of unusual patterns
- **Metrics Monitoring** - Real-time system metrics
- **Alert Management** - Configure and manage log-based alerts
- **Dark Mode UI** - Beautiful cyberpunk-inspired interface

## 📋 Prerequisites

- Node.js 18+ 
- npm or yarn
- Running LogPulse Go backend

## 🛠️ Installation

### 1. Install Dependencies

```bash
npm install
# or
yarn install
```

### 2. Configure Environment (Optional)

```bash
cp .env.example .env.local
# Edit .env.local with your settings
```

### 3. Start Development Server

```bash
npm run dev
# or
yarn dev
```

The app will open at `http://localhost:5173`

## 🔧 Configuration

### Connect to Backend

1. Click the ⚙️ **Settings** icon in the header
2. Enter your backend URL (e.g., `http://localhost:8080`)
3. (Optional) Enter API key if authentication is enabled
4. Click **Test Connection** to verify
5. Click **Save & Connect**

### Quick Presets

The settings panel includes quick presets for common configurations:
- **Local (8080)** - Default Go backend port
- **Local (8081)** - Alternative port
- **Docker (8082)** - Docker Compose setup

## 📖 Usage

### Query Logs

1. Go to the **Logs** tab
2. Enter a query using LogQL syntax:
   ```
   {service="api-gateway", level="error"}
   ```
3. Select time range (5m, 15m, 1h, 6h, 24h)
4. Click **Run Query**

### Live Streaming

1. Go to the **Live** tab
2. Click **Start Live Stream**
3. (Optional) Add filters to limit what you see
4. Use **Pause** to stop scrolling temporarily

### Test Panel

1. Click **Test Panel** in the top right
2. Configure service, environment, level, and message
3. Click **Send Log** to inject a test log
4. Use bulk send buttons (10, 100, 1K) for load testing

### Save Searches

1. Execute a query
2. In the saved searches panel, click **+**
3. Enter a name for your search
4. Click **Save**

Your saved searches appear in the left sidebar for quick access.

### Analytics

1. Go to the **Analytics** tab
2. View log volume trends, error rates, and anomalies
3. Automatic anomaly detection highlights unusual patterns
4. Adjust time range (15m, 1h, 6h, 24h)

### Metrics

1. Go to the **Metrics** tab
2. View real-time Prometheus metrics
3. Auto-refreshes every 5 seconds
4. Shows ingestion rate, storage, and query performance

### Alerts

1. Go to the **Alerts** tab
2. Click **New Rule**
3. Configure:
   - Name and severity
   - Query pattern
   - Threshold and duration
   - Optional webhook URL
4. Click **Create Rule**

## 🏗️ Project Structure

```
src/
├── components/
│   ├── AlertConfig.tsx          # Alert rule management
│   ├── AnalyticsCharts.tsx      # Analytics dashboard
│   ├── ConnectionStatus.tsx     # Connection indicator
│   ├── Header.tsx               # Top navigation
│   ├── LabelsExplorer.tsx       # Label browser
│   ├── LiveStream.tsx           # WebSocket streaming
│   ├── LogExport.tsx            # Export functionality
│   ├── LogViewer.tsx            # Log display
│   ├── MetricsDashboard.tsx     # Metrics display
│   ├── QueryBar.tsx             # Query interface
│   ├── SavedSearches.tsx        # Search management
│   ├── SettingsPanel.tsx        # Backend configuration
│   ├── TestPanel.tsx            # Test log injection
│   └── ui/                      # Reusable UI components
├── hooks/
│   ├── useBackendConnection.ts  # Connection management
│   └── use-mobile.tsx           # Responsive helpers
├── lib/
│   ├── api.ts                   # API client
│   ├── streamClient.ts          # WebSocket client
│   └── utils.ts                 # Utilities
├── pages/
│   ├── Index.tsx                # Main app page
│   └── NotFound.tsx             # 404 page
└── types/
    └── logs.ts                  # TypeScript types
```

## 🎨 Customization

### Theme

The app uses CSS variables for theming. Edit `src/index.css`:

```css
:root {
  --primary: 174 72% 56%;        /* Cyan */
  --success: 142 72% 45%;        /* Green */
  --warning: 38 92% 55%;         /* Orange */
  --destructive: 0 72% 55%;      /* Red */
}
```

### Logo

Replace the logo in `src/components/Header.tsx`.

## 🚢 Production Build

```bash
npm run build
# or
yarn build
```

This creates an optimized production build in the `dist/` directory.

### Deploy to Netlify/Vercel

1. Connect your repository
2. Set build command: `npm run build`
3. Set publish directory: `dist`
4. Deploy!

### Deploy with Docker

```dockerfile
FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

```bash
docker build -t logpulse-frontend .
docker run -p 3000:80 logpulse-frontend
```

## 🐛 Troubleshooting

### Cannot Connect to Backend

**Problem**: "Connection failed" error in settings

**Solutions**:
1. Verify backend is running: `curl http://localhost:8080/health`
2. Check CORS is enabled in backend
3. Ensure URL format is correct: `http://localhost:8080` (with protocol)
4. Check browser console for detailed errors

### WebSocket Connection Fails

**Problem**: Live streaming doesn't work

**Solutions**:
1. Verify WebSocket endpoint: `ws://localhost:8080/stream`
2. Check firewall isn't blocking WebSocket connections
3. Ensure backend `/stream` endpoint is implemented
4. Check browser console for WebSocket errors

### Logs Not Appearing

**Problem**: Query returns 0 results

**Solutions**:
1. Verify logs exist in backend: `curl 'http://localhost:8080/query?query={}&limit=10'`
2. Check time range - logs might be outside selected range
3. Verify query syntax is correct
4. Use Test Panel to inject test logs

### Metrics Not Loading

**Problem**: Metrics dashboard is empty

**Solutions**:
1. Verify `/metrics` endpoint exists
2. Check backend is exposing Prometheus metrics
3. Ensure backend has processed some logs
4. Refresh the metrics dashboard

## 📝 Development Tips

### Hot Module Replacement

Vite provides instant HMR. Changes to components reflect immediately without full page reload.

### Debug Mode

Enable debug logging:
```typescript
// In src/lib/api.ts
const DEBUG = true;
```

### Mock Backend

For frontend development without backend:

```typescript
// src/lib/api-mock.ts
export const mockApiClient = {
  async health() {
    return { status: 'healthy', ingestionRate: 100, ... };
  },
  async query() {
    return { logs: [...], stats: { ... } };
  },
};
```

### Component Testing

```bash
npm run test
# or
yarn test
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## 📄 License

MIT License - see LICENSE file

## 🙏 Acknowledgments

- Built with React + TypeScript + Vite
- UI components from shadcn/ui
- Charts from Recharts
- Icons from Lucide React
- Inspired by Grafana and Kibana

---

**Need help?** Check the [main README](../README.md) or open an issue on GitHub.