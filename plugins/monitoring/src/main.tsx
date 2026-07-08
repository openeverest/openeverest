import type { PluginApi, PluginRegisterFn, ClusterDetailTabProps } from '@openeverest/plugin-sdk';
import { LineChart } from '@mui/x-charts/LineChart';
import { useState, useEffect, useCallback } from 'react';

// @ts-ignore
import { Select, MenuItem, FormControl, InputLabel, Box, CircularProgress, Typography, Grid, Paper } from '@mui/material';

let React: any;
let pluginFetch: (input: string, init?: RequestInit) => Promise<Response>;

type TimeRange = '1h' | '6h' | '24h' | '7d';

const PREDEFINED_METRICS = {
  cpu: { label: 'CPU Usage', query: 'sum(rate(node_cpu_seconds_total{mode!="idle"}[5m])) by (instance)' },
  ram: { label: 'RAM Usage', query: '100 * (1 - ((node_memory_MemFree_bytes + node_memory_Cached_bytes + node_memory_Buffers_bytes) / node_memory_MemTotal_bytes))' },
  disk: { label: 'Disk Usage', query: '100 - (node_filesystem_avail_bytes{fstype=~"ext4|xfs"} / node_filesystem_size_bytes{fstype=~"ext4|xfs"} * 100)' },
};

const MetricChart = ({ namespace, instanceName, metricQuery, metricLabel, timeRange }: { namespace: string, instanceName: string, metricQuery: string, metricLabel: string, timeRange: TimeRange }) => {
  const [data, setData] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState<boolean>(true);

  const fetchMetrics = useCallback(async () => {
    try {
      setLoading(true);
      const now = Math.floor(Date.now() / 1000);
      let start = now;
      let step = 60; // 1m

      if (timeRange === '1h') {
        start = now - 3600;
        step = 60;
      } else if (timeRange === '6h') {
        start = now - 6 * 3600;
        step = 5 * 60;
      } else if (timeRange === '24h') {
        start = now - 24 * 3600;
        step = 30 * 60;
      } else if (timeRange === '7d') {
        start = now - 7 * 24 * 3600;
        step = 6 * 3600;
      }

      const query = encodeURIComponent(metricQuery);
      const res = await pluginFetch(
        `/v1/namespaces/${namespace}/database-clusters/${instanceName}/monitoring/metrics?query=${query}&start=${start}&end=${now}&step=${step}`
      );

      if (!res.ok) {
        throw new Error(`HTTP Error ${res.status}: ${res.statusText}`);
      }

      const json = await res.json();
      setData(json);
      setError(null);
    } catch (err: any) {
      setError(err.message || 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, [namespace, instanceName, metricQuery, timeRange]);

  useEffect(() => {
    fetchMetrics();
  }, [fetchMetrics]);

  if (loading) {
    return React.createElement(Box, { sx: { display: 'flex', justifyContent: 'center', p: 3 } }, 
      React.createElement(CircularProgress, null)
    );
  }

  if (error) {
    return React.createElement(Box, { sx: { color: 'error.main', p: 3 } }, `Failed to load ${metricLabel}: ${error}`);
  }

  const xData: number[] = [];
  const yData: number[] = [];
  
  if (data?.data?.result && data.data.result.length > 0) {
    const values = data.data.result[0].values;
    values.forEach((val: any) => {
      xData.push(val[0] * 1000); // Unix timestamp to ms
      yData.push(parseFloat(val[1]));
    });
  }

  return React.createElement(Paper, { variant: 'outlined', sx: { p: 2, height: '100%' } },
    React.createElement(Typography, { variant: 'h6', gutterBottom: true }, metricLabel),
    xData.length > 0 ? (
      React.createElement(LineChart, {
        xAxis: [{ 
          data: xData,
          scaleType: 'time',
          valueFormatter: (value: number) => new Date(value).toLocaleTimeString()
        }],
        series: [{ 
          data: yData, 
          label: metricLabel,
          showMark: false,
          color: '#8b5cf6'
        }],
        height: 250,
        margin: { left: 50, right: 20, top: 20, bottom: 30 }
      })
    ) : (
      React.createElement(Typography, { color: 'textSecondary' }, 'No metric data available for this range.')
    )
  );
};

const MonitoringTab = (props: ClusterDetailTabProps) => {
  const [timeRange, setTimeRange] = useState<TimeRange>('1h');
  const [customMetric, setCustomMetric] = useState<string>('cpu');

  return React.createElement(Box, { sx: { p: 3 } },
    // Toolbar
    React.createElement(Box, { sx: { display: 'flex', gap: 2, mb: 3, alignItems: 'center' } },
      React.createElement(FormControl, { size: 'small', sx: { minWidth: 120 } },
        React.createElement(InputLabel, null, 'Time Range'),
        React.createElement(Select, {
          value: timeRange,
          label: 'Time Range',
          onChange: (e: any) => setTimeRange(e.target.value as TimeRange)
        },
          React.createElement(MenuItem, { value: '1h' }, 'Last 1 Hour'),
          React.createElement(MenuItem, { value: '6h' }, 'Last 6 Hours'),
          React.createElement(MenuItem, { value: '24h' }, 'Last 24 Hours'),
          React.createElement(MenuItem, { value: '7d' }, 'Last 7 Days')
        )
      ),
      React.createElement(FormControl, { size: 'small', sx: { minWidth: 200 } },
        React.createElement(InputLabel, null, 'Custom Metric'),
        React.createElement(Select, {
          value: customMetric,
          label: 'Custom Metric',
          onChange: (e: any) => setCustomMetric(e.target.value as string)
        },
          React.createElement(MenuItem, { value: 'cpu' }, 'CPU Usage'),
          React.createElement(MenuItem, { value: 'ram' }, 'RAM Usage'),
          React.createElement(MenuItem, { value: 'disk' }, 'Disk Usage'),
          // A truly custom PromQL query could be added here or via a text field,
          // but for the dropdown we provide these options
        )
      )
    ),

    // Grid of predefined charts (Dashboard style)
    React.createElement(Grid, { container: true, spacing: 3, sx: { mb: 4 } },
      React.createElement(Grid, { item: true, xs: 12, md: 6 },
        React.createElement(MetricChart, {
          namespace: props.namespace,
          instanceName: props.instanceName,
          metricLabel: 'CPU Usage',
          metricQuery: PREDEFINED_METRICS.cpu.query,
          timeRange: timeRange
        })
      ),
      React.createElement(Grid, { item: true, xs: 12, md: 6 },
        React.createElement(MetricChart, {
          namespace: props.namespace,
          instanceName: props.instanceName,
          metricLabel: 'RAM Usage',
          metricQuery: PREDEFINED_METRICS.ram.query,
          timeRange: timeRange
        })
      ),
      React.createElement(Grid, { item: true, xs: 12, md: 6 },
        React.createElement(MetricChart, {
          namespace: props.namespace,
          instanceName: props.instanceName,
          metricLabel: 'Disk Usage',
          metricQuery: PREDEFINED_METRICS.disk.query,
          timeRange: timeRange
        })
      ),
      React.createElement(Grid, { item: true, xs: 12, md: 6 },
        React.createElement(MetricChart, {
          namespace: props.namespace,
          instanceName: props.instanceName,
          metricLabel: 'Active Connections (MySQL)',
          metricQuery: 'mysql_global_status_threads_connected',
          timeRange: timeRange
        })
      )
    ),

    // Custom Metric section
    customMetric !== 'cpu' && customMetric !== 'ram' && customMetric !== 'disk' && React.createElement(Box, null,
      React.createElement(Typography, { variant: 'h5', sx: { mb: 2 } }, 'Custom Metric View'),
      React.createElement(MetricChart, {
        namespace: props.namespace,
        instanceName: props.instanceName,
        metricLabel: customMetric,
        metricQuery: customMetric, // In a real app this would be the actual PromQL from a text field
        timeRange: timeRange
      })
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
