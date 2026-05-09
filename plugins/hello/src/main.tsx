import type { PluginRegisterFn, PluginApi, PluginRouteProps } from '@openeverest/plugin-sdk';

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
};

export default register;
