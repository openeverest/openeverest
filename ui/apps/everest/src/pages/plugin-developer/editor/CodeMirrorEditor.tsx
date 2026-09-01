// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { useEffect, useRef } from 'react';
import { Box } from '@mui/material';
import { EditorState, Compartment, Annotation } from '@codemirror/state';
import {
  EditorView,
  keymap,
  lineNumbers,
  highlightActiveLine,
} from '@codemirror/view';
import {
  defaultKeymap,
  history,
  historyKeymap,
  indentWithTab,
} from '@codemirror/commands';
import {
  syntaxHighlighting,
  defaultHighlightStyle,
  indentOnInput,
} from '@codemirror/language';
import { yaml } from '@codemirror/lang-yaml';
import {
  lintGutter,
  setDiagnostics as cmSetDiagnostics,
  type Diagnostic as CmDiagnostic,
} from '@codemirror/lint';
import { oneDark } from '@codemirror/theme-one-dark';
import { Diagnostic, EditorTheme, lineColToOffset } from './types';

// Matches the debounce the previous iframe bundle used, so validation cost per
// keystroke is unchanged.
const CHANGE_DEBOUNCE_MS = 150;

// Marks the doc replace we dispatch from the value-sync effect, so the update
// listener can tell our own programmatic write apart from a genuine user edit.
const ExternalSync = Annotation.define<boolean>();

const themeExtension = (theme: EditorTheme) =>
  theme === 'dark' ? oneDark : syntaxHighlighting(defaultHighlightStyle);

// The app serves a per-response CSP nonce in <meta name="csp-nonce">; Emotion
// reads it the same way in App.tsx. Feeding it to EditorView.cspNonce makes CM6
// stamp its runtime-injected <style> tags with the nonce, so they pass the
// strict style-src instead of being blocked.
const readCspNonce = () =>
  document.querySelector("meta[name='csp-nonce']")?.getAttribute('content') ||
  '';

const toCmDiagnostics = (text: string, diags: Diagnostic[]): CmDiagnostic[] =>
  diags.map((d) => {
    let from = Math.min(lineColToOffset(text, d.line, d.column), text.length);
    let to =
      d.endLine != null && d.endColumn != null
        ? lineColToOffset(text, d.endLine, d.endColumn)
        : from + 1;
    to = Math.min(to, text.length);
    // Ensure a visible, valid range (from <= to, and >= 1 char when the
    // document is non-empty). At end-of-document, walk `from` back one char.
    if (from >= to && text.length > 0) {
      if (from > 0) {
        from -= 1;
        to = from + 1;
      } else {
        to = Math.min(from + 1, text.length);
      }
    }
    return {
      from,
      to: Math.max(to, from),
      severity: d.severity,
      message: d.message,
    };
  });

type Props = {
  value: string;
  onChange: (value: string) => void;
  diagnostics: Diagnostic[];
  theme: EditorTheme;
};

export const CodeMirrorEditor = ({
  value,
  onChange,
  diagnostics,
  theme,
}: Props) => {
  const hostRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const changeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const themeCompartment = useRef(new Compartment());
  // Last text emitted via onChange; lets the value-sync effect skip the stale
  // round-trip of our own debounced edit.
  const lastEmittedRef = useRef<string | null>(null);

  // Mount CodeMirror once. Initial value/theme come from the first render; later
  // prop changes are handled by the effects below.
  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;

    const state = EditorState.create({
      doc: value,
      extensions: [
        EditorView.cspNonce.of(readCspNonce()),
        lineNumbers(),
        highlightActiveLine(),
        history(),
        indentOnInput(),
        lintGutter(),
        yaml(),
        keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
        themeCompartment.current.of(themeExtension(theme)),
        EditorView.updateListener.of((u) => {
          if (!u.docChanged) return;
          // Ignore our own value-sync replace; every real user edit (type, paste,
          // delete, undo, redo) is unmarked and flows through.
          if (u.transactions.some((t) => t.annotation(ExternalSync))) return;
          const text = u.state.doc.toString();
          if (changeTimer.current) clearTimeout(changeTimer.current);
          changeTimer.current = setTimeout(() => {
            lastEmittedRef.current = text;
            onChangeRef.current(text);
          }, CHANGE_DEBOUNCE_MS);
        }),
      ],
    });

    const view = new EditorView({ state, parent: host });
    viewRef.current = view;

    return () => {
      if (changeTimer.current) clearTimeout(changeTimer.current);
      view.destroy();
      viewRef.current = null;
    };
    // Mount-only: props are synced via the dedicated effects below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Sync an external value change (e.g. Format YAML) into the doc. The guard
  // skips our own edits round-tripping back through the parent; comparing
  // against lastEmittedRef also covers the case where `value` lags the live
  // doc by a keystroke while a debounced onChange is still in flight.
  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;
    const current = view.state.doc.toString();
    if (value === current || value === lastEmittedRef.current) return;
    view.dispatch({
      changes: { from: 0, to: current.length, insert: value },
      annotations: ExternalSync.of(true),
    });
  }, [value]);

  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;
    const text = view.state.doc.toString();
    view.dispatch(
      cmSetDiagnostics(view.state, toCmDiagnostics(text, diagnostics))
    );
  }, [diagnostics]);

  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;
    view.dispatch({
      effects: themeCompartment.current.reconfigure(themeExtension(theme)),
    });
  }, [theme]);

  return (
    <Box
      ref={hostRef}
      sx={{
        width: '100%',
        height: '100%',
        '& .cm-editor': { height: '100%' },
        '& .cm-scroller': { overflow: 'auto' },
      }}
    />
  );
};
