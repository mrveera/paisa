import { autocompletion, closeBrackets, completeFromList } from "@codemirror/autocomplete";
import { history } from "@codemirror/commands";
import { bracketMatching, HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { EditorView } from "codemirror";
import { javascript } from "@codemirror/lang-javascript";
import { tags } from "@lezer/highlight";

// Available variables in formulas
const VARIABLES = [
  { label: "amount", type: "variable", info: "Posting amount in default currency" },
  { label: "quantity", type: "variable", info: "Number of units" },
  { label: "date", type: "variable", info: "Posting date" },
  { label: "days_held", type: "variable", info: "Days since posting date" },
  { label: "months_held", type: "variable", info: "Months since posting date" },
  { label: "years_held", type: "variable", info: "Years since posting date" },
  { label: "note", type: "variable", info: "Transaction note" },
  { label: "account", type: "variable", info: "Account name" },
  { label: "commodity", type: "variable", info: "Commodity name" }
];

// Available functions in formulas
const FUNCTIONS = [
  // Interest functions
  {
    label: "simple_interest",
    type: "function",
    info: "simple_interest(principal, annual_rate%, days) → interest amount",
    apply: "simple_interest(amount, 12, days_held)"
  },
  {
    label: "compound_interest",
    type: "function",
    info: "compound_interest(principal, annual_rate%, days, compounds_per_year) → total value",
    apply: "compound_interest(amount, 12, days_held, 12)"
  },
  {
    label: "monthly_interest",
    type: "function",
    info: "monthly_interest(principal, monthly_rate%, days) → interest amount",
    apply: "monthly_interest(amount, 1, days_held)"
  },
  {
    label: "daily_interest",
    type: "function",
    info: "daily_interest(principal, daily_rate%, days) → interest amount",
    apply: "daily_interest(amount, 0.03, days_held)"
  },

  // Note parsing functions
  {
    label: "parse_note_float",
    type: "function",
    info: 'parse_note_float(note, prefix) → extracts float after prefix. e.g., parse_note_float("Int:12.5 Per:M", "Int:") → 12.5',
    apply: 'parse_note_float(note, "Int:")'
  },
  {
    label: "parse_note_string",
    type: "function",
    info: 'parse_note_string(note, prefix) → extracts string after prefix',
    apply: 'parse_note_string(note, "Per:")'
  },
  {
    label: "note_contains",
    type: "function",
    info: "note_contains(note, substr) → true if note contains substr",
    apply: 'note_contains(note, "live")'
  },

  // Math functions
  { label: "min", type: "function", info: "min(a, b) → minimum of two values", apply: "min(amount, 10000)" },
  { label: "max", type: "function", info: "max(a, b) → maximum of two values", apply: "max(amount, 0)" },
  { label: "round", type: "function", info: "round(x) → nearest integer", apply: "round(amount)" },
  { label: "floor", type: "function", info: "floor(x) → round down", apply: "floor(amount)" },
  { label: "ceil", type: "function", info: "ceil(x) → round up", apply: "ceil(amount)" },
  { label: "abs", type: "function", info: "abs(x) → absolute value", apply: "abs(amount)" },
  { label: "pow", type: "function", info: "pow(base, exp) → base^exp", apply: "pow(1.01, months_held)" },
  { label: "sqrt", type: "function", info: "sqrt(x) → square root", apply: "sqrt(amount)" },
  {
    label: "clamp",
    type: "function",
    info: "clamp(value, min, max) → restrict to range",
    apply: "clamp(amount, 0, 100000)"
  },

  // Conditional
  {
    label: "if_else",
    type: "function",
    info: "if_else(condition, trueVal, falseVal) → conditional value",
    apply: "if_else(days_held > 365, amount * 1.1, amount)"
  }
];

// Formula snippets for common use cases
const SNIPPETS = [
  {
    label: "Simple Interest Formula",
    type: "text",
    apply: "amount + simple_interest(amount, parse_note_float(note, \"Rate:\"), days_held)",
    info: "Calculates simple interest based on rate from note"
  },
  {
    label: "Compound Interest Formula",
    type: "text",
    apply: "compound_interest(amount, parse_note_float(note, \"Rate:\"), days_held, 12)",
    info: "Monthly compounding interest based on rate from note"
  },
  {
    label: "P2P Loan Interest",
    type: "text",
    apply: "amount + (amount * parse_note_float(note, \"Int:\") / 100 / 365 * days_held)",
    info: "Annual interest rate from note, calculated daily"
  },
  {
    label: "Tiered Interest",
    type: "text",
    apply: `if_else(days_held > 365,
  amount + simple_interest(amount, 12, days_held),
  amount + simple_interest(amount, 10, days_held)
)`,
    info: "Higher rate after 1 year"
  }
];

const formulaTheme = EditorView.theme({
  "&": {
    fontSize: "13px",
    border: "1px solid var(--border-color)",
    borderRadius: "4px"
  },
  ".cm-content": {
    fontFamily: "monospace",
    minHeight: "80px",
    padding: "8px"
  },
  ".cm-focused": {
    outline: "none",
    borderColor: "var(--primary-color)"
  },
  ".cm-tooltip": {
    maxWidth: "400px"
  },
  ".cm-tooltip-autocomplete": {
    "& > ul > li": {
      padding: "4px 8px"
    }
  },
  ".cm-completionLabel": {
    fontFamily: "monospace"
  },
  ".cm-completionDetail": {
    fontStyle: "normal",
    color: "var(--text-muted)"
  }
});

const formulaHighlight = HighlightStyle.define([
  { tag: tags.keyword, color: "#5c6bc0" },
  { tag: tags.number, color: "#43a047" },
  { tag: tags.string, color: "#e91e63" },
  { tag: tags.comment, color: "#9e9e9e" },
  { tag: tags.function(tags.variableName), color: "#fb8c00" },
  { tag: tags.variableName, color: "#1976d2" },
  { tag: tags.operator, color: "#607d8b" }
]);

export function createFormulaEditor(
  content: string,
  dom: Element,
  onChange: (value: string) => void
) {
  const allCompletions = [...VARIABLES, ...FUNCTIONS, ...SNIPPETS];

  const editor = new EditorView({
    extensions: [
      formulaTheme,
      javascript(),
      syntaxHighlighting(formulaHighlight),
      bracketMatching(),
      closeBrackets(),
      history(),
      EditorView.contentAttributes.of({ "data-enable-grammarly": "false" }),
      autocompletion({
        override: [
          () => ({
            from: 0,
            options: allCompletions
          }),
          (context) => {
            const word = context.matchBefore(/\w*/);
            if (!word || (word.from === word.to && !context.explicit)) {
              return null;
            }
            return {
              from: word.from,
              options: allCompletions
            };
          }
        ],
        activateOnTyping: true,
        defaultKeymap: true
      }),
      EditorView.lineWrapping,
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          onChange(update.state.doc.toString());
        }
      })
    ],
    doc: content,
    parent: dom
  });

  return editor;
}

export function updateContent(editor: EditorView, content: string) {
  if (editor.state.doc.toString() !== content) {
    editor.dispatch({
      changes: { from: 0, to: editor.state.doc.length, insert: content }
    });
  }
}

export function getContent(editor: EditorView): string {
  return editor.state.doc.toString();
}

