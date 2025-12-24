<script lang="ts">
  import { onMount } from "svelte";
  import { ajax, formatCurrency, formatFloat, type Loan, type LoanSummary, type LoanAlert, type LoanStatus } from "$lib/utils";
  import _ from "lodash";

  let loans: Loan[] = [];
  let summary: LoanSummary | null = null;
  let alerts: LoanAlert[] = [];
  let loading = true;
  let filterStatus: LoanStatus | "all" = "all";

  function getStatusKeys(s: LoanSummary): LoanStatus[] {
    return Object.keys(s.by_status) as LoanStatus[];
  }

  onMount(async () => {
    const data = await ajax("/api/loans/dashboard");
    loans = data.loans;
    summary = data.summary;
    alerts = data.alerts;
    loading = false;
  });

  $: filteredLoans = filterStatus === "all" 
    ? loans 
    : loans.filter(l => l.status === filterStatus);

  function getStatusColor(status: LoanStatus): string {
    switch (status) {
      case "overdue": return "is-danger";
      case "maturing": return "is-warning";
      case "active": return "is-success";
      case "closed": return "is-light";
      default: return "is-info";
    }
  }

  function getStatusIcon(status: LoanStatus): string {
    switch (status) {
      case "overdue": return "🔴";
      case "maturing": return "🟡";
      case "active": return "🟢";
      case "closed": return "⚪";
      default: return "🔵";
    }
  }

  function getRiskColor(risk: string): string {
    switch (risk.toLowerCase()) {
      case "high": return "is-danger";
      case "medium": return "is-warning";
      case "low": return "is-success";
      default: return "is-light";
    }
  }

  function formatDays(days: number): string {
    if (days < 0) return `${Math.abs(days)} days overdue`;
    if (days === 0) return "Due today";
    if (days === 1) return "1 day left";
    return `${days} days left`;
  }

  function getAccountName(account: string): string {
    const parts = account.split(":");
    return parts[parts.length - 1];
  }
</script>

<svelte:head>
  <title>Loans | Paisa</title>
</svelte:head>

<section class="section">
  <div class="container is-fluid">
    <!-- Header -->
    <div class="level mb-5">
      <div class="level-left">
        <div class="level-item">
          <h1 class="title is-3">
            <span class="icon-text">
              <span class="icon"><i class="fas fa-hand-holding-usd"></i></span>
              <span>Loans Dashboard</span>
            </span>
          </h1>
        </div>
      </div>
    </div>

    {#if loading}
      <div class="has-text-centered py-6">
        <span class="icon is-large">
          <i class="fas fa-spinner fa-pulse fa-2x"></i>
        </span>
        <p class="mt-3">Loading loans...</p>
      </div>
    {:else if loans.length === 0}
      <div class="notification is-info is-light">
        <p><strong>No loans found.</strong></p>
        <p class="mt-2">
          Loans are detected from accounts matching your custom valuations config with a <code>Target:</code> field in the note.
        </p>
        <p class="mt-2">Example note format: <code>;live Int:2 Per:M Target:3yr Risk:low</code></p>
      </div>
    {:else}
      <!-- Summary Cards -->
      {#if summary}
        <div class="columns is-multiline mb-5">
          <div class="column is-3">
            <div class="box has-background-primary-light">
              <p class="heading">Total Lent</p>
              <p class="title is-4 has-text-primary">{formatCurrency(summary.total_lent)}</p>
            </div>
          </div>
          <div class="column is-3">
            <div class="box has-background-success-light">
              <p class="heading">Current Value</p>
              <p class="title is-4 has-text-success">{formatCurrency(summary.total_value)}</p>
            </div>
          </div>
          <div class="column is-3">
            <div class="box has-background-info-light">
              <p class="heading">Total Gain</p>
              <p class="title is-4 has-text-info">{formatCurrency(summary.total_gain)}</p>
            </div>
          </div>
          <div class="column is-3">
            <div class="box">
              <p class="heading">Accounts</p>
              <p class="title is-4">{summary.total_accounts}</p>
            </div>
          </div>
        </div>

        <!-- Status Summary -->
        <div class="columns mb-5">
          <div class="column is-6">
            <div class="box">
              <h3 class="title is-5 mb-4">By Status</h3>
              <div class="columns is-multiline">
                {#each getStatusKeys(summary) as status}
                  <div class="column is-6">
                    <button 
                      class="button is-fullwidth {filterStatus === status ? getStatusColor(status) : 'is-light'}"
                      on:click={() => filterStatus = filterStatus === status ? "all" : status}
                    >
                      <span>{getStatusIcon(status)} {status}</span>
                      <span class="tag is-rounded ml-2">{summary.by_status[status].count}</span>
                    </button>
                  </div>
                {/each}
              </div>
            </div>
          </div>
          <div class="column is-6">
            <div class="box">
              <h3 class="title is-5 mb-4">By Risk</h3>
              <div class="columns is-multiline">
                {#each Object.entries(summary.by_risk) as [risk, data]}
                  <div class="column is-4">
                    <div class="notification {getRiskColor(risk)} py-2 px-3 mb-0">
                      <p class="is-size-7 has-text-weight-bold">{risk.toUpperCase()}</p>
                      <p class="is-size-6">{data.count} loans</p>
                      <p class="is-size-7">{formatCurrency(data.amount)}</p>
                    </div>
                  </div>
                {/each}
              </div>
            </div>
          </div>
        </div>
      {/if}

      <!-- Alerts -->
      {#if alerts.length > 0}
        <div class="box mb-5 has-background-danger-light">
          <h3 class="title is-5 mb-3">
            <span class="icon-text">
              <span class="icon has-text-danger"><i class="fas fa-exclamation-triangle"></i></span>
              <span>Alerts ({alerts.length})</span>
            </span>
          </h3>
          <div class="content">
            {#each alerts as alert}
              <div class="notification {alert.severity === 'high' ? 'is-danger' : 'is-warning'} is-light py-2 mb-2">
                <div class="columns is-vcentered is-mobile">
                  <div class="column">
                    <strong>{getAccountName(alert.account)}</strong>
                    <span class="ml-2 is-size-7 has-text-grey">{alert.account}</span>
                  </div>
                  <div class="column is-narrow">
                    <span class="tag {alert.severity === 'high' ? 'is-danger' : 'is-warning'}">
                      {alert.message}
                    </span>
                  </div>
                  <div class="column is-narrow">
                    <strong>{formatCurrency(alert.amount)}</strong>
                  </div>
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/if}

      <!-- Loans Table -->
      <div class="box">
        <div class="level mb-4">
          <div class="level-left">
            <h3 class="title is-5 mb-0">
              All Loans 
              {#if filterStatus !== "all"}
                <span class="tag {getStatusColor(filterStatus)} ml-2">{filterStatus}</span>
                <button class="delete is-small ml-1" on:click={() => filterStatus = "all"}></button>
              {/if}
            </h3>
          </div>
          <div class="level-right">
            <span class="has-text-grey">Showing {filteredLoans.length} of {loans.length} loans</span>
          </div>
        </div>

        <div class="table-container">
          <table class="table is-fullwidth is-hoverable">
            <thead>
              <tr>
                <th>Account</th>
                <th class="has-text-right">Principal</th>
                <th class="has-text-right">Current Value</th>
                <th class="has-text-right">Gain</th>
                <th class="has-text-centered">Rate</th>
                <th class="has-text-centered">Maturity</th>
                <th class="has-text-centered">Status</th>
                <th class="has-text-centered">Risk</th>
              </tr>
            </thead>
            <tbody>
              {#each filteredLoans as loan}
                <tr>
                  <td>
                    <a href="/assets/gain/{encodeURIComponent(loan.account)}">
                      <strong>{getAccountName(loan.account)}</strong>
                    </a>
                    <br>
                    <span class="is-size-7 has-text-grey">{loan.account}</span>
                  </td>
                  <td class="has-text-right">{formatCurrency(loan.principal)}</td>
                  <td class="has-text-right has-text-success">{formatCurrency(loan.current_value)}</td>
                  <td class="has-text-right has-text-info">+{formatCurrency(loan.gain_amount)}</td>
                  <td class="has-text-centered">
                    {formatFloat(loan.interest_rate, 1)}%
                    <span class="is-size-7 has-text-grey">/{loan.period || 'Y'}</span>
                  </td>
                  <td class="has-text-centered">
                    {#if loan.maturity_date}
                      <span class="is-size-7">
                        {loan.maturity_date.format("MMM YYYY")}
                      </span>
                      <br>
                      <span class="is-size-7 has-text-grey">
                        {formatDays(loan.days_to_maturity)}
                      </span>
                      {#if loan.percent_complete > 0}
                        <progress 
                          class="progress is-small {loan.status === 'overdue' ? 'is-danger' : loan.status === 'maturing' ? 'is-warning' : 'is-success'} mt-1" 
                          value={Math.min(loan.percent_complete, 100)} 
                          max="100"
                        ></progress>
                      {/if}
                    {:else}
                      <span class="has-text-grey">-</span>
                    {/if}
                  </td>
                  <td class="has-text-centered">
                    <span class="tag {getStatusColor(loan.status)}">
                      {getStatusIcon(loan.status)} {loan.status}
                    </span>
                  </td>
                  <td class="has-text-centered">
                    <span class="tag {getRiskColor(loan.risk_level)}">
                      {loan.risk_level}
                    </span>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/if}
  </div>
</section>

<style>
  .box {
    border-radius: 8px;
  }
  
  .progress.is-small {
    height: 4px;
  }
  
  .table td {
    vertical-align: middle;
  }
</style>

