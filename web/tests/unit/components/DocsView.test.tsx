import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DocsView } from '../../../src/components/DocsView';
import type { DocIndex } from 'go-ui';

// A minimal DocIndex the stubbed fetch returns for DocsApp's doc.json request.
const DOC_INDEX: DocIndex = {
  module: 'github.com/malcolmston/streamlit',
  packages: [
    {
      importPath: 'github.com/malcolmston/streamlit/st',
      name: 'st',
      synopsis: 'Package st is a standard-library-only Go port of Python Streamlit.',
      doc: 'Package st is a standard-library-only Go port of Python Streamlit.',
      consts: [],
      vars: [],
      types: [
        {
          name: 'Session',
          signature: 'type Session struct{}',
          doc: 'Session represents a single browser connection to the app.',
          consts: [],
          vars: [],
          funcs: [],
          methods: [],
        },
      ],
      funcs: [{ name: 'Run', signature: 'func Run(app func(*Session), addr string) error', doc: 'Run starts an HTTP server that serves the app.' }],
    },
  ],
};

describe('DocsView', () => {
  beforeEach(() => {
    // DocsApp fetches doc.json; return the small index.
    global.fetch = vi.fn((input: RequestInfo | URL) => {
      if (String(input).includes('doc.json')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(DOC_INDEX) } as Response);
      }
      return new Promise<Response>(() => {});
    }) as unknown as typeof fetch;
  });

  it('renders the inline React API reference from the fetched doc.json', async () => {
    const { container } = render(<DocsView />);
    expect(container.querySelector('#view-docs')).not.toBeNull();
    expect(
      screen.getByRole('heading', { level: 2, name: /API documentation/ }),
    ).toBeInTheDocument();

    // DocsApp fetches asynchronously, then renders the package view + symbols.
    expect(await screen.findByRole('heading', { name: /package st/ })).toBeInTheDocument();
    expect(container.querySelector('#sym-Run'), 'func Run symbol card').not.toBeNull();
    expect(container.querySelector('#sym-Session'), 'type Session symbol card').not.toBeNull();

    // The secondary link to the raw generated static HTML remains.
    expect(screen.getByRole('link', { name: /Open the raw generated HTML/ })).toHaveAttribute('href', './api/');
  });
});
