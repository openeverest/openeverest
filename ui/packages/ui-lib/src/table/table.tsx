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

import FilterListIcon from '@mui/icons-material/FilterList';
import KeyboardDoubleArrowDownIcon from '@mui/icons-material/KeyboardDoubleArrowDown';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import SearchIcon from '@mui/icons-material/Search';
import ViewColumnIcon from '@mui/icons-material/ViewColumn';
import { Alert, AlertProps, Box } from '@mui/material';
import {
  MaterialReactTable,
  MRT_ShowHideColumnsButton,
  MRT_VisibilityState,
  MRT_ToggleFiltersButton,
  MRT_ToggleGlobalFilterButton,
} from 'material-react-table';
import { useEffect } from 'react';
import { ICONS_OPACITY } from './table.constants';
import { TableProps } from './table.types';
import usePersistentColumnVisibility from './usePersistentColumnVisibility';

const NoDataAlertMessage = ({
  message,
  ...rest
}: { message: string } & AlertProps) => {
  const { sx, ...alertProps } = rest;
  return (
    <Alert
      severity="info"
      sx={[
        {
          width: '100%',
          alignItems: 'center',
          marginTop: 1,
          marginBottom: 1,
          height: '60px',
        },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
      {...alertProps}
    >
      {message}
    </Alert>
  );
};

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function Table<T extends Record<string, any>>(props: TableProps<T>) {
  const {
    data,
    columns,
    muiTablePaperProps,
    muiTopToolbarProps,
    displayColumnDefOptions,
    noDataMessage = 'No data',
    noDataAlertProps,
    emptyFilterResultsMessage = 'No data found',
    hideExpandAllIcon,
    tableName,
    state,
    initialState,
    emptyState,
    enableRowHoverAction = false,
    rowHoverAction = () => {},
    muiTableBodyRowProps,
    getRowId,
    ...rest
  } = props;
  const [columnVisibility, setColumnVisibility] =
    usePersistentColumnVisibility(tableName);

  let columnVisibilityState: MRT_VisibilityState | undefined = {};
  let restOfState = {};

  if (state) {
    const { columnVisibility: cv, ...rest } = state;
    columnVisibilityState = cv;
    restOfState = rest;
  }

  const stopPropagation = (e: Event) => {
    e.stopPropagation();
  };

  // While loading with no data, MRT renders fake skeleton rows whose columns
  // are all `null`. A consumer `getRowId` that reads those columns then returns
  // the same id for every skeleton row (e.g. "null/null"), producing duplicate
  // React keys. React can no longer reconcile those rows away once real data
  // arrives, leaving phantom skeleton rows stuck forever. Give skeleton rows a
  // unique index-based id instead, matching MRT's own skeleton condition.
  const showingSkeletonRows =
    !!(state?.isLoading || state?.showSkeletons) && !data.length;
  const safeGetRowId: typeof getRowId = getRowId
    ? (originalRow, index, parentRow) =>
        showingSkeletonRows
          ? `mrt-skeleton-row-${index}`
          : getRowId(originalRow, index, parentRow)
    : undefined;

  // @ts-expect-error
  const { sx: muiTopToolbarPropsSx = {}, ...muiTopToolbarRestProps } =
    muiTopToolbarProps || {};

  useEffect(() => {
    const hideColumnsIcon = document.querySelector(
      '[aria-label="Show/Hide columns"]'
    );
    const showFiltesIcon = document.querySelector(
      '[aria-label="Show/Hide filters"]'
    );
    const globalFilterIcon = document.querySelector(
      '[aria-label="Show/Hide search"]'
    );
    const elementsWithExpandLabel = document.querySelectorAll(
      '[aria-label="Column Actions"]'
    );

    if (!data.length) {
      hideColumnsIcon?.addEventListener('click', stopPropagation);
      showFiltesIcon?.addEventListener('click', stopPropagation);
      globalFilterIcon?.addEventListener('click', stopPropagation);
      elementsWithExpandLabel.forEach((element) => {
        element.addEventListener('click', stopPropagation);
      });
    }

    return () => {
      globalFilterIcon?.removeEventListener('click', stopPropagation);
      showFiltesIcon?.removeEventListener('click', stopPropagation);
      hideColumnsIcon?.removeEventListener('click', stopPropagation);
      elementsWithExpandLabel.forEach((element) => {
        element.removeEventListener('click', stopPropagation);
      });
    };
  }, [data]);

  // disable hiding for first 2 columns
  const customColumns = columns.map((col, index) => {
    if (index < 2) {
      return { ...col, enableHiding: false };
    }
    return col;
  });

  return (
    <MaterialReactTable
      renderEmptyRowsFallback={({ table: { getPreFilteredRowModel } }) => (
        <>
          {/* This means there was data before filtering, so we show the message of empty filtering result */}
          {getPreFilteredRowModel().rows.length > 0 ? (
            <NoDataAlertMessage
              message={emptyFilterResultsMessage}
              {...noDataAlertProps}
            />
          ) : emptyState ? (
            <Box>{emptyState}</Box>
          ) : (
            <NoDataAlertMessage message={noDataMessage} {...noDataAlertProps} />
          )}
        </>
      )}
      layoutMode="grid"
      enablePagination={data.length > 10}
      enableBottomToolbar={data.length > 10}
      enableDensityToggle={false}
      enableFullScreenToggle={false}
      enableSorting={!!data.length}
      autoResetAll={false}
      onColumnVisibilityChange={setColumnVisibility}
      icons={{
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        KeyboardDoubleArrowDownIcon: (propsIcon: any) =>
          data.length ? (
            <KeyboardDoubleArrowDownIcon {...propsIcon} />
          ) : (
            <KeyboardDoubleArrowDownIcon sx={{ opacity: ICONS_OPACITY }} />
          ),
        SearchIcon: () => (
          <SearchIcon sx={{ opacity: !data.length ? ICONS_OPACITY : 1 }} />
        ),
        FilterListIcon: () => (
          <FilterListIcon sx={{ opacity: !data.length ? ICONS_OPACITY : 1 }} />
        ),
        ViewColumnIcon: () => (
          <ViewColumnIcon sx={{ opacity: !data.length ? ICONS_OPACITY : 1 }} />
        ),
        MoreVertIcon: () => (
          <MoreVertIcon sx={{ opacity: !data.length ? ICONS_OPACITY : 1 }} />
        ),
      }}
      positionActionsColumn="last"
      positionExpandColumn="last"
      muiTablePaperProps={{ elevation: 0, ...muiTablePaperProps }}
      muiTopToolbarProps={{
        sx: {
          backgroundColor: 'transparent',
          '& > .MuiBox-root': {
            flexDirection: 'row-reverse',
            flexWrap: 'wrap',

            '& > .MuiBox-root:has(.percona-table-internal-actions)': {
              marginRight: 'auto',
            },
          },
          ...muiTopToolbarPropsSx,
        },
        ...muiTopToolbarRestProps,
      }}
      renderToolbarInternalActions={({ table }) => (
        <Box className="percona-table-internal-actions">
          <MRT_ToggleGlobalFilterButton table={table} />
          <MRT_ToggleFiltersButton table={table} />
          <MRT_ShowHideColumnsButton table={table} />
        </Box>
      )}
      displayColumnDefOptions={{
        'mrt-row-actions': {
          size: 30,
          muiTableBodyCellProps: {
            sx: {
              flex: 'none',
              width: '75px',
              ...// @ts-ignore
              // prettier-ignore
              displayColumnDefOptions?.['mrt-row-actions']
                // @ts-ignore
                ?.muiTableBodyCellProps?.sx,
            },
            ...displayColumnDefOptions?.['mrt-row-actions']
              ?.muiTableBodyCellProps,
          },
          muiTableHeadCellProps: {
            sx: {
              flex: 'none',
              width: '75px',
              // We could simply set "mrt-row-actions.header" to ""
              // However, MRT takes that string and shows it in the show/hide columns menu
              // By doing this, we still have "Actions" in that menu, but no text (i.e. transparent) in the header cell
              color: 'transparent',
              userSelect: 'none',
              WebkitUserSelect: 'none',
              MozUserSelect: 'none',
              msUserSelect: 'none',
              ...// @ts-ignore
              // prettier-ignore
              displayColumnDefOptions?.['mrt-row-actions']
              // @ts-ignore
                ?.muiTableHeadCellProps?.sx,
            },
            ...displayColumnDefOptions?.['mrt-row-actions']
              ?.muiTableHeadCellProps,
          },
          ...displayColumnDefOptions?.['mrt-row-actions'],
        },
        'mrt-row-expand': {
          size: 40,
          muiTableBodyCellProps: {
            sx: {
              flex: 'none',
              width: '60px',
              ...// @ts-ignore
              // prettier-ignore
              displayColumnDefOptions?.['mrt-row-expand']?.muiTableBodyCellProps
                // @ts-ignore
                ?.sx,
            },
            ...displayColumnDefOptions?.['mrt-row-expand']
              ?.muiTableBodyCellProps,
          },
          muiTableHeadCellProps: {
            sx: {
              flex: 'none',
              width: '60px',
              ...(!!hideExpandAllIcon && {
                '& button': {
                  display: 'none',
                },
              }),
              ...// @ts-ignore
              // prettier-ignore
              displayColumnDefOptions?.['mrt-row-expand']?.muiTableHeadCellProps
              // @ts-ignore
                ?.sx,
            },
            ...displayColumnDefOptions?.['mrt-row-expand']
              ?.muiTableHeadCellProps,
          },
          ...displayColumnDefOptions?.['mrt-row-expand'],
        },
        ...displayColumnDefOptions,
      }}
      muiTableProps={{
        // @ts-expect-error
        'data-testid': tableName,
      }}
      muiTableHeadProps={{
        sx: {
          '& tr': {
            backgroundColor: 'transparent',
          },
        },
      }}
      muiTableBodyProps={{
        sx: {
          '& tr': {
            backgroundColor: 'transparent',
          },
        },
      }}
      muiDetailPanelProps={{
        sx: {
          width: '100%',
        },
      }}
      {...rest}
      columns={customColumns}
      data={data}
      getRowId={safeGetRowId}
      state={{
        columnVisibility: { ...columnVisibility, ...columnVisibilityState },
        ...restOfState,
      }}
      initialState={{
        columnVisibility,
        ...initialState,
      }}
      muiTableBodyRowProps={(args) => {
        const { row, isDetailPanel } = args;
        const ownProps =
          (typeof muiTableBodyRowProps === 'function'
            ? muiTableBodyRowProps(args)
            : muiTableBodyRowProps) || {};
        const { sx, onClick, ...restOfProps } = ownProps;
        return {
          onClick: (e) => {
            if (
              !isDetailPanel &&
              enableRowHoverAction &&
              e.currentTarget.contains(e.target as Node)
            ) {
              rowHoverAction(row);
              onClick?.(e);
            }
          },
          sx: [
            !isDetailPanel &&
              enableRowHoverAction && {
                cursor: 'pointer', // you might want to change the cursor too when adding an onClick
              },
            ...(Array.isArray(sx) ? sx : [sx]),
          ],
          ...restOfProps,
        };
      }}
    />
  );
}

export default Table;
