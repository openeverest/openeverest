import type { PluginRegisterFn, PluginApi, PluginRouteProps } from '@openeverest/plugin-sdk';

// The plugin receives React from the host via the register API
// to avoid duplicate React instances and bare-specifier import issues.
let React: PluginApi['React'];

const HelloPage = (props: PluginRouteProps) => {
  const [count, setCount] = React.useState(0);

  return React.createElement('div', { style: { padding: '2rem' } },
    React.createElement('h1', null, '👋 Hello from Plugin!'),
    React.createElement('p', null,
      'This page is served by a dynamically loaded plugin module running inside OpenEverest.',
    ),
    React.createElement('p', { style: { color: '#666' } },
      `Plugin: ${props.pluginName}`,
    ),
    React.createElement('button', {
      onClick: () => setCount((c) => c + 1),
      style: { padding: '0.5rem 1rem', fontSize: '1rem', cursor: 'pointer' },
    }, `Clicked ${count} times`),
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
