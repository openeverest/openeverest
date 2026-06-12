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

import { useContext, useEffect, useRef, useState } from 'react';
import UpgradeEverestContext from './upgrade-everest.context';
import { useVersion } from 'hooks';
import { AuthContext } from 'contexts/auth';

const UpgradeEverestProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const commitVersion = useRef<null | string>(null);
  const { authStatus } = useContext(AuthContext);
  const { data: apiVersion } = useVersion({
    enabled: authStatus === 'loggedIn',
  });

  const [currentVersion, setCurrentVersion] = useState('');

  const [openReloadEverestDialog, setOpenReloadEverestDialog] = useState(false);

  useEffect(() => {
    if (commitVersion.current === null && apiVersion?.fullCommit) {
      commitVersion.current = apiVersion?.fullCommit;
      setCurrentVersion(apiVersion?.version);
    }
    if (
      commitVersion.current !== null &&
      commitVersion.current !== apiVersion?.fullCommit
    ) {
      commitVersion.current = apiVersion?.fullCommit ?? commitVersion.current;
      setOpenReloadEverestDialog(true);
    }
  }, [apiVersion?.fullCommit, apiVersion?.version]);

  const toggleOpenReloadDialog = () =>
    setOpenReloadEverestDialog((val) => !val);

  return (
    <UpgradeEverestContext.Provider
      value={{
        openReloadDialog: openReloadEverestDialog,
        toggleOpenReloadDialog,
        setOpenReloadDialog: setOpenReloadEverestDialog,
        currentVersion: currentVersion,
        apiVersion: apiVersion?.version,
      }}
    >
      {children}
    </UpgradeEverestContext.Provider>
  );
};

export default UpgradeEverestProvider;
