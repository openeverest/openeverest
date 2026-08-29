/* eslint-disable no-console */
// Must run before any plugin bundle is dynamically imported.
import './plugin-runtime-host';
import ReactDOM from 'react-dom/client';
import App from 'App';

// We don't use SSR, so we suppress this SSR annoying waring about first-child
const consoleError = console.error;

console.error = function filterErrors(msg, ...args) {
  if (/server-side rendering/.test(msg)) {
    return;
  }
  consoleError(msg, ...args);
};

ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
