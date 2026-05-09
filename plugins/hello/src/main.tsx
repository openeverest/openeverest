import type {
  PluginRegisterFn,
  PluginApi,
  PluginRouteProps,
  ClusterDetailTabProps,
  ClusterActionProps,
  ClusterCardProps,
  GlobalDashboardWidgetProps,
  SettingsPanelProps,
} from '@openeverest/plugin-sdk';

// The plugin receives React from the host via the register API
// to avoid duplicate React instances and bare-specifier import issues.
let React: PluginApi['React'];

interface EverestEvent {
  resourceVersion: string;
  type: string;
  occurredAt: string;
  namespace: string;
  resource: { kind: string; name: string; uid: string };
  newState?: { phase?: string };
}

const EventFeed = () => {
  const [events, setEvents] = React.useState<EverestEvent[]>([]);
  const [connected, setConnected] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    const token = localStorage.getItem('everestToken');
    const url = '/v1/events';

    // EventSource doesn't support custom headers, so we use fetch + ReadableStream.
    const controller = new AbortController();

    (async () => {
      try {
        const resp = await fetch(url, {
          headers: { Authorization: `Bearer ${token}` },
          signal: controller.signal,
        });
        if (!resp.ok || !resp.body) {
          setError(`Failed to connect: ${resp.status}`);
          return;
        }
        setConnected(true);
        setError(null);

        const reader = resp.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });

          // Parse SSE frames from buffer.
          const frames = buffer.split('\n\n');
          buffer = frames.pop() || '';

          for (const frame of frames) {
            const dataLine = frame.split('\n').find((l) => l.startsWith('data: '));
            if (!dataLine) continue;
            try {
              const evt: EverestEvent = JSON.parse(dataLine.slice(6));
              setEvents((prev) => [evt, ...prev].slice(0, 50));
            } catch {
              // skip malformed frames
            }
          }
        }
      } catch (err: unknown) {
        if (err instanceof Error && err.name !== 'AbortError') {
          setError(err.message);
        }
      } finally {
        setConnected(false);
      }
    })();

    return () => controller.abort();
  }, []);

  return React.createElement('div', { style: { marginTop: '1.5rem' } },
    React.createElement('h2', { style: { marginBottom: '0.5rem' } }, 'Live Event Stream'),
    React.createElement('p', { style: { color: connected ? '#2e7d32' : '#999' } },
      connected ? '● Connected to /v1/events' : error ? `● Error: ${error}` : '○ Connecting…',
    ),
    events.length === 0
      ? React.createElement('p', { style: { color: '#999', fontStyle: 'italic' } },
          'Waiting for events… Try creating or deleting a database cluster.',
        )
      : React.createElement('table', {
          style: { width: '100%', borderCollapse: 'collapse', fontSize: '0.875rem' },
        },
          React.createElement('thead', null,
            React.createElement('tr', { style: { textAlign: 'left', borderBottom: '2px solid #ddd' } },
              React.createElement('th', { style: { padding: '0.5rem' } }, 'Time'),
              React.createElement('th', { style: { padding: '0.5rem' } }, 'Type'),
              React.createElement('th', { style: { padding: '0.5rem' } }, 'Resource'),
              React.createElement('th', { style: { padding: '0.5rem' } }, 'Namespace'),
              React.createElement('th', { style: { padding: '0.5rem' } }, 'Phase'),
            ),
          ),
          React.createElement('tbody', null,
            ...events.map((evt, i) =>
              React.createElement('tr', {
                key: evt.resourceVersion + i,
                style: { borderBottom: '1px solid #eee' },
              },
                React.createElement('td', { style: { padding: '0.5rem', fontFamily: 'monospace' } },
                  new Date(evt.occurredAt).toLocaleTimeString(),
                ),
                React.createElement('td', { style: { padding: '0.5rem' } }, evt.type),
                React.createElement('td', { style: { padding: '0.5rem' } },
                  `${evt.resource.kind}/${evt.resource.name}`,
                ),
                React.createElement('td', { style: { padding: '0.5rem' } }, evt.namespace),
                React.createElement('td', { style: { padding: '0.5rem' } },
                  evt.newState?.phase || '—',
                ),
              ),
            ),
          ),
        ),
  );
};

const HelloPage = (props: PluginRouteProps) => {
  return React.createElement('div', { style: { padding: '2rem' } },
    React.createElement('h1', null, '👋 Hello from Plugin!'),
    React.createElement('p', null,
      'This page is served by a dynamically loaded plugin module running inside OpenEverest.',
    ),
    React.createElement('p', { style: { color: '#666' } },
      `Plugin: ${props.pluginName}`,
    ),
    React.createElement(EventFeed, null),
  );
};

// --- Phase 4 extension point demos ---

const HelloClusterTab = (props: ClusterDetailTabProps) => {
  return React.createElement('div', { style: { padding: '1rem' } },
    React.createElement('h3', null, '👋 Hello Tab'),
    React.createElement('p', null, `Instance: ${props.instanceName}`),
    React.createElement('p', null, `Namespace: ${props.namespace}`),
    React.createElement('pre', { style: { fontSize: '0.75rem', background: '#f5f5f5', padding: '0.5rem', borderRadius: 4, overflow: 'auto', maxHeight: 200 } },
      JSON.stringify(props.cluster, null, 2),
    ),
  );
};

const HelloClusterAction = (props: ClusterActionProps) => {
  return React.createElement('div', { style: { padding: '1rem' } },
    React.createElement('h3', null, '👋 Hello Action'),
    React.createElement('p', null, `Namespace: ${props.namespace}`),
    React.createElement('p', null, 'This dialog was triggered from the cluster actions menu.'),
    React.createElement('button', {
      onClick: props.onClose,
      style: { marginTop: '1rem', padding: '0.5rem 1rem', cursor: 'pointer' },
    }, 'Close'),
  );
};

const HelloClusterCard = (props: ClusterCardProps) => {
  return React.createElement('div', null,
    React.createElement('p', { style: { margin: 0 } },
      `Namespace: ${props.namespace}`,
    ),
    React.createElement('p', { style: { margin: '0.25rem 0', color: '#666', fontSize: '0.875rem' } },
      'This card is contributed by the Hello plugin.',
    ),
  );
};

const HelloDashboardWidget = (props: GlobalDashboardWidgetProps) => {
  return React.createElement('div', null,
    React.createElement('p', { style: { margin: 0 } },
      `Accessible namespaces: ${props.namespaces.length}`,
    ),
    React.createElement('ul', { style: { margin: '0.5rem 0', paddingLeft: '1.25rem' } },
      ...props.namespaces.map((ns) =>
        React.createElement('li', { key: ns, style: { fontSize: '0.875rem' } }, ns),
      ),
    ),
  );
};

const HelloSettingsPanel = (_props: SettingsPanelProps) => {
  return React.createElement('div', { style: { padding: '1rem' } },
    React.createElement('h3', null, '👋 Hello Settings'),
    React.createElement('p', null, 'This settings tab is contributed by the Hello plugin.'),
    React.createElement('p', { style: { color: '#666' } },
      'Plugin settings and configuration would go here.',
    ),
  );
};

const register: PluginRegisterFn = (api: PluginApi) => {
  React = api.React;

  api.registerExtension({
    type: 'sidebarItem',
    label: 'Hello Plugin',
  });

  api.registerExtension({
    type: 'route',
    label: 'Hello Plugin',
    component: HelloPage,
  });

  api.registerExtension({
    type: 'clusterDetailTab',
    label: 'Hello',
    path: 'hello',
    component: HelloClusterTab,
  });

  api.registerExtension({
    type: 'clusterAction',
    label: 'Hello Action',
    component: HelloClusterAction,
  });

  api.registerExtension({
    type: 'clusterCard',
    label: 'Hello Card',
    component: HelloClusterCard,
  });

  api.registerExtension({
    type: 'globalDashboardWidget',
    label: 'Hello Widget',
    component: HelloDashboardWidget,
  });

  api.registerExtension({
    type: 'settingsPanel',
    label: 'Hello',
    path: 'hello',
    component: HelloSettingsPanel,
  });
};

export default register;
