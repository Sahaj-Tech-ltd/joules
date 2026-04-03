<script lang="ts">
  import { marked } from 'marked';

  let { text = '' }: { text?: string } = $props();

  let html = $derived.by(() => {
    if (!text) return '';
    const result = marked.parse(text, { async: false, breaks: true }) as string;
    return result;
  });
</script>

<div class="prose prose-sm max-w-none">
  {@html html}
</div>

<style>
  :global(.prose) {
    line-height: 1.625;
  }
  :global(.prose :where(strong)) {
    color: var(--color-foreground);
    font-weight: 600;
  }
  :global(.prose :where(ul)) {
    list-style-type: disc;
    margin: 0.5em 0;
    padding-left: 1.5em;
  }
  :global(.prose :where(ol)) {
    list-style-type: decimal;
    margin: 0.5em 0;
    padding-left: 1.5em;
  }
  :global(.prose :where(li)) {
    margin: 0.15em 0;
  }
  :global(.prose :where(p)) {
    margin: 0.4em 0;
  }
  :global(.prose :where(p:first-child)) {
    margin-top: 0;
  }
  :global(.prose :where(p:last-child)) {
    margin-bottom: 0;
  }
  :global(.prose :where(code)) {
    background: var(--color-accent);
    padding: 0.15em 0.35em;
    border-radius: 0.25em;
    font-size: 0.875em;
    color: var(--color-foreground);
  }
  :global(.prose :where(a)) {
    color: var(--color-primary);
    text-decoration: underline;
  }
</style>
