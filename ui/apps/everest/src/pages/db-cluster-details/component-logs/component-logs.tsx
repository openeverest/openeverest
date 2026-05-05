// everest
// Copyright (C) 2023 Percona LLC
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

import { useState, useMemo, useEffect, useRef } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import {
  Box,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  Paper,
  Typography,
  Alert,
  IconButton,
  Tooltip,
} from '@mui/material';
import DownloadIcon from '@mui/icons-material/Download';
import ContentCopyOutlinedIcon from '@mui/icons-material/ContentCopyOutlined';
import { useDbClusterComponents } from 'hooks/api/db-cluster/useDbClusterComponents';
import { useDbClusterComponentLogsStream } from 'hooks/api/db-cluster/useDbClusterComponentLogsStream';
import {
  COMPONENT_STATUS,
  CONTAINER_STATUS,
} from '../components/components.constants';

const Logs = () => {
  const { dbClusterName = '', namespace = '' } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const { data: components = [], isLoading: isLoadingComponents } =
    useDbClusterComponents(namespace, dbClusterName!);

  const filteredComponents = useMemo(() => {
    return components.filter((c) => c.status !== COMPONENT_STATUS.PENDING);
  }, [components]);

  const componentFromUrl = searchParams.get('component') || '';
  const containerFromUrl = searchParams.get('container') || '';

  const [selectedComponent, setSelectedComponent] =
    useState<string>(componentFromUrl);
  const [selectedContainer, setSelectedContainer] =
    useState<string>(containerFromUrl);
  const [copySuccess, setCopySuccess] = useState(false);

  const containers = useMemo(() => {
    const component = filteredComponents.find(
      (c) => c.name === selectedComponent
    );
    return component?.containers || [];
  }, [filteredComponents, selectedComponent]);

  const filteredContainers = useMemo(() => {
    return containers.filter((c) => c.status !== CONTAINER_STATUS.WAITING);
  }, [containers]);

  useEffect(() => {
    const componentToSelect =
      componentFromUrl ||
      (filteredComponents.length > 0 ? filteredComponents[0].name : '');

    if (componentToSelect && componentToSelect !== selectedComponent) {
      setSelectedComponent(componentToSelect);
      if (!componentFromUrl) {
        setSearchParams({ component: componentToSelect });
      }
    }

    const containerToSelect =
      containerFromUrl ||
      (filteredContainers.length > 0 ? filteredContainers[0].name : '');
    setSelectedContainer(containerToSelect);
  }, [
    componentFromUrl,
    filteredComponents,
    selectedComponent,
    containerFromUrl,
    filteredContainers,
    setSearchParams,
  ]);

  const shouldFetchLogs =
    !!selectedComponent &&
    (filteredContainers.length === 0 || !!selectedContainer);

  const {
    logs,
    isConnecting,
    error: logsError,
    getFullLogs,
  } = useDbClusterComponentLogsStream(
    namespace,
    dbClusterName!,
    selectedComponent,
    selectedContainer || undefined,
    {
      enabled: shouldFetchLogs,
    }
  );

  const logsContainerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (logsContainerRef.current) {
      logsContainerRef.current.scrollTop =
        logsContainerRef.current.scrollHeight;
    }
  }, [logs]);

  const handleDownload = async () => {
    if (!selectedComponent) return;

    const fullLogs = await getFullLogs();
    const blob = new Blob([fullLogs], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const fileName = `${dbClusterName}-${selectedComponent}${selectedContainer ? `-${selectedContainer}` : ''}-logs.txt`;

    const link = document.createElement('a');
    link.href = url;
    link.download = fileName;
    link.click();
    URL.revokeObjectURL(url);
  };

  const handleCopy = async () => {
    if (!selectedComponent) return;

    const fullLogs = await getFullLogs();
    if (navigator.clipboard) {
      await navigator.clipboard.writeText(fullLogs);
      setCopySuccess(true);
      setTimeout(() => setCopySuccess(false), 1500);
    }
  };

  return (
    <Stack gap={2} sx={{ mt: 2 }}>
      <Stack direction="row" gap={2} sx={{ flexWrap: 'wrap' }}>
        {filteredComponents.length > 0 && (
          <FormControl sx={{ minWidth: 250 }}>
            <InputLabel id="component-select-label">Component</InputLabel>
            <Select
              id="component-select"
              value={selectedComponent}
              label="Component"
              onChange={(e) => {
                const newComponent = e.target.value;
                setSelectedComponent(newComponent);
                setSelectedContainer('');
                setSearchParams((prev) => {
                  const newParams = new URLSearchParams(prev);
                  newParams.set('component', newComponent);
                  newParams.delete('container');
                  return newParams;
                });
              }}
              disabled={isLoadingComponents}
            >
              {filteredComponents.map((component) => (
                <MenuItem key={component.name} value={component.name}>
                  {component.name} ({component.type})
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        )}

        {selectedComponent && filteredContainers.length > 0 && (
          <FormControl sx={{ minWidth: 250 }}>
            <InputLabel id="container-select-label">Container</InputLabel>
            <Select
              id="container-select"
              value={selectedContainer}
              label="Container"
              onChange={(e) => {
                const newContainer = e.target.value;
                setSelectedContainer(newContainer);
                setSearchParams((prev) => {
                  const newParams = new URLSearchParams(prev);
                  newContainer
                    ? newParams.set('container', newContainer)
                    : newParams.delete('container');
                  return newParams;
                });
              }}
            >
              {filteredContainers.map((container) => (
                <MenuItem key={container.name} value={container.name}>
                  {container.name}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        )}
      </Stack>

      {!selectedComponent &&
        (filteredComponents.length > 0 ? (
          <Alert severity="info">Please select a component to view logs</Alert>
        ) : (
          <Alert severity="info">No components available</Alert>
        ))}

      {selectedComponent && (
        <Paper
          sx={{
            p: 2,
            backgroundColor: 'surfaces.elevation0',
            minHeight: 150,
            maxHeight: '50vh',
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: 1,
              mb: 1,
            }}
          >
            {logs && (
              <Box sx={{ display: 'flex', gap: 1, ml: 'auto' }}>
                <Tooltip title={copySuccess ? 'Copied!' : 'Copy logs'}>
                  <IconButton
                    size="small"
                    onClick={handleCopy}
                    disabled={!selectedComponent}
                    aria-label="Copy logs"
                  >
                    <ContentCopyOutlinedIcon />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Download logs">
                  <IconButton
                    size="small"
                    onClick={handleDownload}
                    disabled={!selectedComponent}
                    aria-label="Download logs"
                  >
                    <DownloadIcon />
                  </IconButton>
                </Tooltip>
              </Box>
            )}
          </Box>
          <Box
            ref={logsContainerRef}
            sx={{
              flex: 1,
              overflowY: 'auto',
              overflowX: 'auto',
            }}
          >
            {isConnecting ? (
              <Typography color="text.secondary">...</Typography>
            ) : logsError ? (
              <Alert severity="error">
                Failed to load logs. Please try again.
              </Alert>
            ) : logs ? (
              <Typography
                component="pre"
                sx={{
                  fontFamily: 'monospace',
                  fontSize: '0.875rem',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                  margin: 0,
                  color: 'text.primary',
                }}
              >
                {logs}
              </Typography>
            ) : (
              <Typography color="text.secondary">No logs available</Typography>
            )}
          </Box>
        </Paper>
      )}
    </Stack>
  );
};

export default Logs;
