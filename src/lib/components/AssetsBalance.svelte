<script lang="ts">
  import { type AssetBreakdown, buildTree, isZero } from "$lib/utils";
  import _ from "lodash";
  import Table from "./Table.svelte";
  import type { ColumnDefinition } from "tabulator-tables";
  import {
    accountName,
    formatCurrencyChange,
    indendedAssetAccountName,
    nonZeroCurrency,
    nonZeroFloatChange,
    nonZeroPercentageChange
  } from "$lib/table_formatters";
  import { showZeroValueAccounts } from "../../persisted_store";
  import { refresh } from "../../store";

  export let breakdowns: Record<string, AssetBreakdown>;
  export let indent = true;

  const columns: ColumnDefinition[] = [
    {
      title: "Account",
      field: "group",
      formatter: indent ? indendedAssetAccountName : accountName,
      frozen: true
    },
    {
      title: "Investment Amount",
      field: "investmentAmount",
      hozAlign: "right",
      vertAlign: "middle",
      formatter: nonZeroCurrency
    },
    {
      title: "Withdrawal Amount",
      field: "withdrawalAmount",
      hozAlign: "right",
      formatter: nonZeroCurrency
    },
    {
      title: "Balance Units",
      field: "balanceUnits",
      hozAlign: "right",
      formatter: nonZeroCurrency
    },
    { title: "Market Value", field: "marketAmount", hozAlign: "right", formatter: nonZeroCurrency },
    { title: "Change", field: "gainAmount", hozAlign: "right", formatter: formatCurrencyChange },
    { title: "XIRR", field: "xirr", hozAlign: "right", formatter: nonZeroFloatChange },
    {
      title: "Absolute Return",
      field: "absoluteReturn",
      hozAlign: "right",
      formatter: nonZeroPercentageChange
    }
  ];

  let tree: AssetBreakdown[] = [];
  let filteredBreakdowns: Record<string, AssetBreakdown> = {};

  $: {
    if (breakdowns) {
      // Filter out zero value accounts if the toggle is off
      filteredBreakdowns = $showZeroValueAccounts
        ? breakdowns
        : _.pickBy(breakdowns, (breakdown) => {
            return !isZero(breakdown.marketAmount) || !isZero(breakdown.balanceUnits);
          });

      tree = buildTree(Object.values(filteredBreakdowns), (i) => i.group);
    }
  }

  function toggleZeroValueAccounts() {
    showZeroValueAccounts.update(value => !value);
    refresh();
  }
</script>

<div class="mb-3 is-flex is-justify-content-flex-end">
  <button class="button is-small" on:click={toggleZeroValueAccounts}>
    <span class="icon is-small">
      <i class="fas {$showZeroValueAccounts ? 'fa-eye-slash' : 'fa-eye'}"></i>
    </span>
    <span>{$showZeroValueAccounts ? 'Hide' : 'Show'} Zero Value Accounts</span>
  </button>
</div>

{#if indent}
  <Table data={tree} tree {columns} />
{:else}
  <Table data={Object.values(filteredBreakdowns)} {columns} />
{/if}
