import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QuickStart } from '../../../src/components/QuickStart';
import { STREAMLIT } from '../../../src/data';

describe('QuickStart', () => {
  it('renders the Quick start heading and highlighted Go snippet', () => {
    const { container } = render(<QuickStart lib={STREAMLIT} />);
    expect(container.querySelector(`#${STREAMLIT.id}-quick`)).not.toBeNull();
    expect(screen.getByRole('heading', { name: 'Quick start' })).toBeInTheDocument();
    // The snippet mentions st.Run.
    expect(container.textContent).toContain('st.Run');
  });
});
