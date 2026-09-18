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

import { useState, useRef, useEffect, useCallback, useMemo } from 'react';

const DRAFT_KEY = 'everest.playground.draft';
const SAVED_KEY = 'everest.playground.saved';
const DRAFT_DEBOUNCE_MS = 500;

type SavedSchemas = Record<string, string>;

export type SchemaStorage = {
  initialDraft: string;
  names: string[];
  saveDraft: (yaml: string) => void;
  saveSchema: (name: string, yaml: string) => void;
  loadSchema: (name: string) => string | undefined;
  deleteSchema: (name: string) => void;
};

// Values are shape-checked, not just parsed: a malformed entry that slips
// through JSON.parse (`null`, a number, an array) would otherwise crash render.
const readInitialDraft = (fallbackYaml: string): string => {
  try {
    const raw = localStorage.getItem(DRAFT_KEY);
    if (raw === null) {
      return fallbackYaml;
    }
    const parsed: unknown = JSON.parse(raw);
    return typeof parsed === 'string' ? parsed : fallbackYaml;
  } catch {
    return fallbackYaml;
  }
};

const readSavedSchemas = (): SavedSchemas => {
  try {
    const raw = localStorage.getItem(SAVED_KEY);
    if (raw === null) {
      return {};
    }
    const parsed: unknown = JSON.parse(raw);
    if (
      parsed === null ||
      typeof parsed !== 'object' ||
      Array.isArray(parsed)
    ) {
      return {};
    }
    return Object.fromEntries(
      Object.entries(parsed).filter(
        (entry): entry is [string, string] => typeof entry[1] === 'string'
      )
    );
  } catch {
    return {};
  }
};

const writeDraft = (yaml: string) => {
  try {
    localStorage.setItem(DRAFT_KEY, JSON.stringify(yaml));
  } catch {
    // Storage unavailable (e.g. private mode); the draft won't persist.
  }
};

const writeSavedSchemas = (saved: SavedSchemas) => {
  try {
    localStorage.setItem(SAVED_KEY, JSON.stringify(saved));
  } catch {
    // Storage unavailable (e.g. private mode); the save won't persist.
  }
};

// Owns the playground's localStorage: a debounced editor draft plus a named
// schema library. Every access is guarded, so unavailable storage degrades to
// in-memory behaviour instead of throwing.
export const useSchemaStorage = (fallbackYaml: string): SchemaStorage => {
  const [initialDraft] = useState(() => readInitialDraft(fallbackYaml));
  const [saved, setSaved] = useState<SavedSchemas>(readSavedSchemas);
  const names = useMemo(() => Object.keys(saved).sort(), [saved]);

  // Updated synchronously by the mutators below, so several calls batched into
  // one render still compose instead of overwriting each other.
  const savedRef = useRef(saved);

  const draftTimeout = useRef<ReturnType<typeof setTimeout>>();
  const pendingDraft = useRef<string>();

  // Flush rather than drop the pending write, otherwise navigating away within
  // the debounce window loses the developer's most recent edits.
  useEffect(
    () => () => {
      if (draftTimeout.current === undefined) {
        return;
      }
      clearTimeout(draftTimeout.current);
      if (pendingDraft.current !== undefined) {
        writeDraft(pendingDraft.current);
      }
    },
    []
  );

  const saveDraft = useCallback((yaml: string) => {
    pendingDraft.current = yaml;
    if (draftTimeout.current !== undefined) {
      clearTimeout(draftTimeout.current);
    }
    draftTimeout.current = setTimeout(() => {
      writeDraft(yaml);
      draftTimeout.current = undefined;
      pendingDraft.current = undefined;
    }, DRAFT_DEBOUNCE_MS);
  }, []);

  const saveSchema = useCallback((name: string, yaml: string) => {
    const next = { ...savedRef.current, [name]: yaml };
    savedRef.current = next;
    writeSavedSchemas(next);
    setSaved(next);
  }, []);

  const deleteSchema = useCallback((name: string) => {
    const next = { ...savedRef.current };
    delete next[name];
    savedRef.current = next;
    writeSavedSchemas(next);
    setSaved(next);
  }, []);

  const loadSchema = useCallback(
    (name: string): string | undefined => savedRef.current[name],
    []
  );

  return {
    initialDraft,
    names,
    saveDraft,
    saveSchema,
    loadSchema,
    deleteSchema,
  };
};
