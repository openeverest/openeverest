import type { PluginApi, PluginRegisterFn, ClusterDetailTabProps } from '@openeverest/plugin-sdk';
import { LineChart } from '@mui/x-charts/LineChart';
import { useState, useEffect } from 'react';

// The plugin's React instance is injected by the host
let React: any;
let pluginFetch: (input: string, init?: RequestInit) => Promise<Response>;

const MonitoringTab = (props: ClusterDetailTabProps) => {
  const [data, setData] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState<boolean>(true);

  useEffect(() => {
    const fetchMetrics = async () => {
      try {
        setLoading(true);
        // We will fetch CPU usage for the last 1 hour
        const now = Math.floor(Date.now() / 1000);
        const start = now - 3600; // 1 hour ago
        const step = 60; // 1 minute resolution

        // Example PromQL query for CPU usage (depends on the monitoring stack, assuming standard node_exporter for PoC)
        const query = encodeURIComponent(`sum(rate(node_cpu_seconds_total{mode!="idle"}[5m])) by (instance)`);

        const res = await pluginFetch(
          `/v1/namespaces/${props.namespace}/database-clusters/${props.instanceName}/monitoring/metrics?query=${query}&start=${start}&end=${now}&step=${step}`
        );

        if (!res.ok) {
          throw new Error(`Failed to fetch metrics: ${res.statusText}`);
        }

        const json = await res.json();
        setData(json);
      } catch (err: any) {
        setError(err.message || 'Unknown error');
      } finally {
        setLoading(false);
      }
    };

    fetchMetrics();
  }, [props.namespace, props.instanceName]);

  if (loading) {
    return React.createElement('div', { style: { padding: '2rem' } }, 'Loading metrics...');
  }

  if (error) {
    return React.createElement('div', { style: { padding: '2rem', color: 'red' } }, `Error: ${error}`);
  }

  // Parse Prometheus response for @mui/x-charts
  const xData: number[] = [];
  const yData: number[] = [];
  
  if (data?.data?.result && data.data.result.length > 0) {
    const values = data.data.result[0].values;
    values.forEach((val: any) => {
      xData.push(val[0] * 1000); // Convert Unix timestamp to ms
      yData.push(parseFloat(val[1]));
    });
  }

  return React.createElement(
    'div',
    { style: { padding: '1rem', width: '100%', height: '400px' } },
    React.createElement('h3', null, 'CPU Usage (Last 1 Hour)'),
    xData.length > 0 ? (
      React.createElement(LineChart, {
        xAxis: [{ 
          data: xData,
          scaleType: 'time',
          valueFormatter: (value: number) => new Date(value).toLocaleTimeString()
        }],
        series: [{ 
          data: yData, 
          label: 'CPU Usage',
          showMark: false,
          color: '#8b5cf6'
        }],
        height: 300,
        margin: { left: 50, right: 20, top: 20, bottom: 30 }
      })
    ) : (
      React.createElement('p', null, 'No metric data available.')
    )
  );
};

const register: PluginRegisterFn = (api: PluginApi) => {
  React = api.React;
  pluginFetch = api.fetch.bind(api);

  api.registerExtension({
    type: 'clusterDetailTab',
    label: 'Monitoring',
    path: 'monitoring',
    component: MonitoringTab,
  });
};

export default register;
