<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { createFormulaEditor, updateContent } from "$lib/formula_editor";
  import type { EditorView } from "codemirror";
  import { ajax } from "$lib/utils";
  import _ from "lodash";

  export let value: string = "";
  export let disabled: boolean = false;

  let editorContainer: HTMLElement;
  let editor: EditorView;
  let previewResult: number | null = null;
  let previewError: string | null = null;
  let isValidating = false;

  // Sample data for preview
  let sampleAmount = 10000;
  let sampleDaysHeld = 30;
  let sampleNote = "Int:12 Per:M";
  let showPreview = false;

  onMount(() => {
    if (!disabled && editorContainer) {
      try {
        editor = createFormulaEditor(value || "", editorContainer, (newValue) => {
          value = newValue;
          debouncedPreview();
        });
      } catch (e) {
        console.error("Failed to create formula editor:", e);
      }
    }
  });

  onDestroy(() => {
    if (editor) {
      editor.destroy();
    }
  });

  $: if (editor && editor.state && value !== editor.state.doc.toString()) {
    try {
      updateContent(editor, value || "");
    } catch (e) {
      console.error("Failed to update formula editor:", e);
    }
  }

  const debouncedPreview = _.debounce(async () => {
    if (!value || !showPreview) return;
    await runPreview();
  }, 500);

  async function runPreview() {
    if (!value) {
      previewResult = null;
      previewError = null;
      return;
    }

    isValidating = true;
    try {
      const response = await ajax("/api/valuations/preview", {
        method: "POST",
        body: JSON.stringify({
          formula: value,
          amount: sampleAmount,
          days_held: sampleDaysHeld,
          note: sampleNote
        })
      });

      if (response.preview?.error) {
        previewError = response.preview.error;
        previewResult = null;
      } else {
        previewResult = response.preview?.result;
        previewError = null;
      }
    } catch (e) {
      previewError = e.message;
      previewResult = null;
    } finally {
      isValidating = false;
    }
  }

  function togglePreview() {
    showPreview = !showPreview;
    if (showPreview) {
      runPreview();
    }
  }
</script>

<div class="formula-editor-container">
  <div class="formula-editor-header">
    <span class="formula-label">Formula</span>
    <button class="button is-small" on:click={togglePreview} type="button">
      <span class="icon is-small">
        <i class="fas {showPreview ? 'fa-eye-slash' : 'fa-eye'}"></i>
      </span>
      <span>{showPreview ? "Hide" : "Show"} Preview</span>
    </button>
  </div>

  {#if disabled}
    <pre class="formula-disabled">{value}</pre>
  {:else}
    <div class="formula-editor" bind:this={editorContainer}></div>
  {/if}

  {#if showPreview}
    <div class="formula-preview">
      <div class="preview-header">
        <span class="icon is-small"><i class="fas fa-flask"></i></span>
        <span>Live Preview</span>
      </div>

      <div class="preview-inputs">
        <div class="field is-horizontal">
          <div class="field-label is-small">
            <label class="label">Amount</label>
          </div>
          <div class="field-body">
            <input
              class="input is-small"
              type="number"
              bind:value={sampleAmount}
              on:change={runPreview}
            />
          </div>
        </div>

        <div class="field is-horizontal">
          <div class="field-label is-small">
            <label class="label">Days Held</label>
          </div>
          <div class="field-body">
            <input
              class="input is-small"
              type="number"
              bind:value={sampleDaysHeld}
              on:change={runPreview}
            />
          </div>
        </div>

        <div class="field is-horizontal">
          <div class="field-label is-small">
            <label class="label">Note</label>
          </div>
          <div class="field-body">
            <input
              class="input is-small"
              type="text"
              bind:value={sampleNote}
              on:change={runPreview}
              placeholder="e.g., Int:12 Per:M"
            />
          </div>
        </div>
      </div>

      <div class="preview-result">
        {#if isValidating}
          <span class="has-text-grey">
            <span class="icon is-small"><i class="fas fa-spinner fa-spin"></i></span>
            Calculating...
          </span>
        {:else if previewError}
          <span class="has-text-danger">
            <span class="icon is-small"><i class="fas fa-exclamation-triangle"></i></span>
            {previewError}
          </span>
        {:else if previewResult !== null}
          <div class="result-display">
            <span class="result-label">Result:</span>
            <span class="result-value has-text-success">
              {previewResult.toLocaleString(undefined, { maximumFractionDigits: 2 })}
            </span>
            <span class="result-change">
              ({((previewResult / sampleAmount - 1) * 100).toFixed(2)}% change)
            </span>
          </div>
        {:else}
          <span class="has-text-grey">Enter a formula to see preview</span>
        {/if}
      </div>
    </div>
  {/if}

  <div class="formula-help">
    <details>
      <summary>
        <span class="icon is-small"><i class="fas fa-circle-question"></i></span>
        Formula Help
      </summary>
      <div class="help-content">
        <p><strong>Variables:</strong> <code>amount</code>, <code>quantity</code>, <code>days_held</code>, <code>months_held</code>, <code>years_held</code>, <code>note</code></p>
        <p><strong>Interest Functions:</strong></p>
        <ul>
          <li><code>simple_interest(principal, annual_rate%, days)</code></li>
          <li><code>compound_interest(principal, annual_rate%, days, compounds_per_year)</code></li>
          <li><code>monthly_interest(principal, monthly_rate%, days)</code></li>
        </ul>
        <p><strong>Note Parsing:</strong></p>
        <ul>
          <li><code>parse_note_float(note, "Int:")</code> - Extract number after prefix</li>
          <li><code>note_contains(note, "live")</code> - Check if note contains text</li>
        </ul>
        <p><strong>Math:</strong> <code>min</code>, <code>max</code>, <code>round</code>, <code>floor</code>, <code>ceil</code>, <code>abs</code>, <code>pow</code>, <code>sqrt</code>, <code>clamp</code></p>
        <p><strong>Conditional:</strong> <code>if_else(condition, trueVal, falseVal)</code></p>
      </div>
    </details>
  </div>
</div>

<style>
  .formula-editor-container {
    width: 100%;
    max-width: 500px;
  }

  .formula-editor-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
  }

  .formula-label {
    font-weight: 500;
    font-size: 0.875rem;
  }

  .formula-editor {
    border-radius: 4px;
    overflow: hidden;
  }

  .formula-disabled {
    background: var(--background-secondary);
    padding: 0.75rem;
    border-radius: 4px;
    font-family: monospace;
    font-size: 0.875rem;
    white-space: pre-wrap;
    word-break: break-word;
  }

  .formula-preview {
    margin-top: 1rem;
    padding: 1rem;
    background: var(--background-secondary);
    border-radius: 4px;
    border: 1px solid var(--border-color);
  }

  .preview-header {
    font-weight: 600;
    margin-bottom: 0.75rem;
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .preview-inputs {
    display: grid;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .preview-inputs .field {
    margin-bottom: 0;
  }

  .preview-inputs .input {
    max-width: 200px;
  }

  .preview-result {
    padding: 0.75rem;
    background: var(--background-primary);
    border-radius: 4px;
  }

  .result-display {
    display: flex;
    align-items: baseline;
    gap: 0.5rem;
  }

  .result-label {
    font-weight: 500;
  }

  .result-value {
    font-size: 1.25rem;
    font-weight: 600;
    font-family: monospace;
  }

  .result-change {
    font-size: 0.875rem;
    color: var(--text-muted);
  }

  .formula-help {
    margin-top: 0.75rem;
  }

  .formula-help summary {
    cursor: pointer;
    font-size: 0.875rem;
    color: var(--text-muted);
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .formula-help summary:hover {
    color: var(--text-primary);
  }

  .help-content {
    margin-top: 0.75rem;
    padding: 0.75rem;
    background: var(--background-secondary);
    border-radius: 4px;
    font-size: 0.8125rem;
  }

  .help-content p {
    margin-bottom: 0.5rem;
  }

  .help-content ul {
    margin-left: 1.5rem;
    margin-bottom: 0.5rem;
  }

  .help-content code {
    background: var(--background-primary);
    padding: 0.125rem 0.25rem;
    border-radius: 3px;
    font-size: 0.75rem;
  }
</style>

