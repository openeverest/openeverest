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
import { EditorState } from '@codemirror/state';
import { EditorView, keymap } from '@codemirror/view';
import { defaultKeymap, indentWithTab } from '@codemirror/commands';
import { yaml } from '@codemirror/lang-yaml';
import {
  syntaxHighlighting,
  defaultHighlightStyle,
  indentOnInput,
  bracketMatching,
} from '@codemirror/language';
import { getCspNonce } from 'utils/csp';

export interface UseYamlEditorOptions {
  /** Initial YAML content */
  initialContent?: string;
  /** Called on every document change */
  onChange?: (value: string) => void;
}

/**
 * Manages a CodeMirror 6 editor instance with YAML syntax highlighting
 * and CSP-safe nonce propagation.
 *
 * CSP architecture:
 * CodeMirror 6 dynamically injects `<style>` elements for its themes
 * and UI chrome. Under a strict nonce-based CSP (which OpenEverest uses),
 * these injections would be blocked. The `EditorView.cspNonce` facet
 * tells CodeMirror to set the `nonce` attribute on all dynamically
 * created `<style>` elements, keeping them CSP-compliant.
 *
 * This hook handles:
 * - Editor creation and teardown (no memory leaks)
 * - Nonce extraction from the server-injected meta tag
 * - YAML language mode with syntax highlighting
 * - Keyboard shortcuts (including Tab for indentation)
 */
export const useYamlEditor = ({
  initialContent = '',
  onChange,
}: UseYamlEditorOptions = {}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const onChangeRef = useRef(onChange);

  // Keep onChange ref current without re-creating the editor
  useEffect(() => {
    onChangeRef.current = onChange;
  }, [onChange]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const nonce = getCspNonce();

    const updateListener = EditorView.updateListener.of((update) => {
      if (update.docChanged && onChangeRef.current) {
        onChangeRef.current(update.state.doc.toString());
      }
    });

    const state = EditorState.create({
      doc: initialContent,
      extensions: [
        // CSP nonce — CRITICAL for production CSP compliance.
        // Without this, CodeMirror's dynamic <style> injections are
        // blocked by the nonce-based style-src policy.
        EditorView.cspNonce.of(nonce),
        yaml(),
        syntaxHighlighting(defaultHighlightStyle),
        indentOnInput(),
        bracketMatching(),
        keymap.of([...defaultKeymap, indentWithTab]),
        updateListener,
        EditorView.lineWrapping,
      ],
    });

    const view = new EditorView({
      state,
      parent: container,
    });

    viewRef.current = view;

    return () => {
      view.destroy();
      viewRef.current = null;
    };
    // Intentionally omit initialContent — we only create the editor once.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return { containerRef, viewRef };
};
