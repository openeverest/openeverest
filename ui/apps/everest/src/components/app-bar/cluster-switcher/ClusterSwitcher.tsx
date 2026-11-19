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

import AddIcon from '@mui/icons-material/Add';
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  TextField,
} from '@mui/material';
import { registerClusterFn } from 'api/clusters';
import { useClusters } from 'contexts/clusters/useClusters';
import { enqueueSnackbar } from 'notistack';
import { FormEvent, useState } from 'react';

type RegistrationForm = {
  name: string;
  server: string;
  bearerToken: string;
  caPem: string;
};

const emptyRegistration: RegistrationForm = {
  name: '',
  server: '',
  bearerToken: '',
  caPem: '',
};

const toBase64 = (value: string) =>
  btoa(String.fromCharCode(...new TextEncoder().encode(value)));

const ClusterSwitcher = () => {
  const { clusterName, clusters, selectCluster, refreshClusters } =
    useClusters();
  const [open, setOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [form, setForm] = useState<RegistrationForm>(emptyRegistration);

  const register = async (event: FormEvent) => {
    event.preventDefault();
    setSubmitting(true);
    try {
      const registered = await registerClusterFn({
        name: form.name,
        server: form.server,
        bearerToken: form.bearerToken,
        ...(form.caPem ? { caData: toBase64(form.caPem) } : {}),
      });
      await refreshClusters();
      selectCluster(registered.name);
      setForm(emptyRegistration);
      setOpen(false);
      enqueueSnackbar(`Cluster ${registered.name} connected`, {
        variant: 'success',
      });
    } catch (error) {
      enqueueSnackbar(
        error instanceof Error ? error.message : 'Failed to connect cluster',
        { variant: 'error' }
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <>
      <Stack direction="row" spacing={1} alignItems="center">
        <FormControl size="small" sx={{ minWidth: 180 }}>
          <InputLabel id="cluster-select-label" sx={{ color: 'inherit' }}>
            Cluster
          </InputLabel>
          <Select
            labelId="cluster-select-label"
            label="Cluster"
            value={clusterName}
            onChange={(event) => selectCluster(event.target.value)}
            sx={{ color: 'inherit' }}
          >
            {clusters.map((cluster) => (
              <MenuItem value={cluster.name} key={cluster.name}>
                {cluster.name}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <Button
          color="inherit"
          startIcon={<AddIcon />}
          onClick={() => setOpen(true)}
        >
          Connect
        </Button>
      </Stack>

      <Dialog
        open={open}
        onClose={() => setOpen(false)}
        component="form"
        onSubmit={register}
        fullWidth
        maxWidth="sm"
      >
        <DialogTitle>Connect Kubernetes cluster</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <TextField
              label="Cluster name"
              value={form.name}
              onChange={(event) =>
                setForm({ ...form, name: event.target.value })
              }
              required
            />
            <TextField
              label="Agent HTTPS URL"
              type="url"
              placeholder="https://everest-agent.example.com"
              value={form.server}
              onChange={(event) =>
                setForm({ ...form, server: event.target.value })
              }
              required
            />
            <TextField
              label="Agent bearer token"
              type="password"
              value={form.bearerToken}
              onChange={(event) =>
                setForm({ ...form, bearerToken: event.target.value })
              }
              required
            />
            <TextField
              label="Agent CA certificate (optional PEM)"
              value={form.caPem}
              onChange={(event) =>
                setForm({ ...form, caPem: event.target.value })
              }
              multiline
              minRows={4}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Cancel</Button>
          <Button type="submit" variant="contained" disabled={submitting}>
            {submitting ? 'Connecting...' : 'Connect cluster'}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
};

export default ClusterSwitcher;
