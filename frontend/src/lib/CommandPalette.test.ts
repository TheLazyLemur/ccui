import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, fireEvent, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';
import CommandPalette from './CommandPalette.svelte';

describe('CommandPalette', () => {
  afterEach(() => {
    cleanup();
  });

  describe('rendering', () => {
    it('renders modal overlay', () => {
      const { container } = render(CommandPalette);
      const overlay = container.querySelector('[data-testid="palette-overlay"]');
      expect(overlay).toBeTruthy();
    });

    it('renders search input', () => {
      const { container } = render(CommandPalette);
      const input = container.querySelector('input[data-testid="palette-input"]');
      expect(input).toBeTruthy();
    });

    it('renders all options by default', () => {
      const { container } = render(CommandPalette);
      const options = container.querySelectorAll('[data-testid^="palette-option-"]');
      expect(options.length).toBe(4);
    });

    it('input has autofocus for focus on mount', () => {
      const { container } = render(CommandPalette);
      const input = container.querySelector('input[data-testid="palette-input"]') as HTMLInputElement;
      expect(input.hasAttribute('autofocus')).toBe(true);
    });
  });

  describe('filtering', () => {
    it('filters options by search query', async () => {
      const { container } = render(CommandPalette);
      const input = container.querySelector('input[data-testid="palette-input"]') as HTMLInputElement;

      await fireEvent.input(input, { target: { value: 'chat' } });
      await tick();

      const options = container.querySelectorAll('[data-testid^="palette-option-"]');
      expect(options.length).toBe(1);
      expect(options[0].getAttribute('data-testid')).toBe('palette-option-chat');
    });

    it('fuzzy matches options', async () => {
      const { container } = render(CommandPalette);
      const input = container.querySelector('input[data-testid="palette-input"]') as HTMLInputElement;

      await fireEvent.input(input, { target: { value: 'term' } });
      await tick();

      const options = container.querySelectorAll('[data-testid^="palette-option-"]');
      expect(options.length).toBe(1);
      expect(options[0].getAttribute('data-testid')).toBe('palette-option-terminal');
    });

    it('shows no options when no match', async () => {
      const { container } = render(CommandPalette);
      const input = container.querySelector('input[data-testid="palette-input"]') as HTMLInputElement;

      await fireEvent.input(input, { target: { value: 'xyz' } });
      await tick();

      const options = container.querySelectorAll('[data-testid^="palette-option-"]');
      expect(options.length).toBe(0);
    });

    it('case insensitive filtering', async () => {
      const { container } = render(CommandPalette);
      const input = container.querySelector('input[data-testid="palette-input"]') as HTMLInputElement;

      await fireEvent.input(input, { target: { value: 'CHAT' } });
      await tick();

      const options = container.querySelectorAll('[data-testid^="palette-option-"]');
      expect(options.length).toBe(1);
    });
  });

  describe('keyboard navigation', () => {
    it('first option selected by default', () => {
      const { container } = render(CommandPalette);
      const options = container.querySelectorAll('[data-testid^="palette-option-"]');
      expect(options[0].classList.contains('selected')).toBe(true);
    });

    it('ArrowDown moves selection down', async () => {
      const { container } = render(CommandPalette);
      const input = container.querySelector('input[data-testid="palette-input"]') as HTMLInputElement;

      await fireEvent.keyDown(input, { key: 'ArrowDown' });
      await tick();

      const options = container.querySelectorAll('[data-testid^="palette-option-"]');
      expect(options[0].classList.contains('selected')).toBe(false);
      expect(options[1].classList.contains('selected')).toBe(true);
    });

    it('ArrowUp moves selection up', async () => {
      const { container } = render(CommandPalette);
      const input = container.querySelector('input[data-testid="palette-input"]') as HTMLInputElement;

      await fireEvent.keyDown(input, { key: 'ArrowDown' });
      await tick();
      await fireEvent.keyDown(input, { key: 'ArrowUp' });
      await tick();

      const options = container.querySelectorAll('[data-testid^="palette-option-"]');
      expect(options[0].classList.contains('selected')).toBe(true);
    });

    it('ArrowDown wraps to top', async () => {
      const { container } = render(CommandPalette);
      const input = container.querySelector('input[data-testid="palette-input"]') as HTMLInputElement;

      // Move to last option (4 options total)
      for (let i = 0; i < 4; i++) {
        await fireEvent.keyDown(input, { key: 'ArrowDown' });
        await tick();
      }

      const options = container.querySelectorAll('[data-testid^="palette-option-"]');
      expect(options[0].classList.contains('selected')).toBe(true);
    });

    it('ArrowUp wraps to bottom', async () => {
      const { container } = render(CommandPalette);
      const input = container.querySelector('input[data-testid="palette-input"]') as HTMLInputElement;

      await fireEvent.keyDown(input, { key: 'ArrowUp' });
      await tick();

      const options = container.querySelectorAll('[data-testid^="palette-option-"]');
      expect(options[3].classList.contains('selected')).toBe(true);
    });

    it('Enter confirms selection', async () => {
      const mockHandler = vi.fn();
      const { container, component } = render(CommandPalette);
      component.$on('select', mockHandler);

      const input = container.querySelector('input[data-testid="palette-input"]') as HTMLInputElement;
      await fireEvent.keyDown(input, { key: 'Enter' });
      await tick();

      expect(mockHandler).toHaveBeenCalledTimes(1);
      expect(mockHandler.mock.calls[0][0].detail).toBe('chat');
    });

    it('Escape emits close event', async () => {
      const mockHandler = vi.fn();
      const { container, component } = render(CommandPalette);
      component.$on('close', mockHandler);

      const input = container.querySelector('input[data-testid="palette-input"]') as HTMLInputElement;
      await fireEvent.keyDown(input, { key: 'Escape' });
      await tick();

      expect(mockHandler).toHaveBeenCalledTimes(1);
    });
  });

  describe('mouse interaction', () => {
    it('click on option emits select', async () => {
      const mockHandler = vi.fn();
      const { container, component } = render(CommandPalette);
      component.$on('select', mockHandler);

      const option = container.querySelector('[data-testid="palette-option-review"]') as HTMLElement;
      await fireEvent.click(option);
      await tick();

      expect(mockHandler).toHaveBeenCalledTimes(1);
      expect(mockHandler.mock.calls[0][0].detail).toBe('review');
    });

    it('click on background emits close', async () => {
      const mockHandler = vi.fn();
      const { container, component } = render(CommandPalette);
      component.$on('close', mockHandler);

      const overlay = container.querySelector('[data-testid="palette-overlay"]') as HTMLElement;
      await fireEvent.click(overlay);
      await tick();

      expect(mockHandler).toHaveBeenCalledTimes(1);
    });

    it('click on modal content does not emit close', async () => {
      const mockHandler = vi.fn();
      const { container, component } = render(CommandPalette);
      component.$on('close', mockHandler);

      const modal = container.querySelector('[data-testid="palette-modal"]') as HTMLElement;
      await fireEvent.click(modal);
      await tick();

      expect(mockHandler).not.toHaveBeenCalled();
    });

    it('hover over option updates selection', async () => {
      const { container } = render(CommandPalette);
      const option = container.querySelector('[data-testid="palette-option-terminal"]') as HTMLElement;

      await fireEvent.mouseEnter(option);
      await tick();

      expect(option.classList.contains('selected')).toBe(true);
    });
  });

  describe('option values', () => {
    it('Chat option emits chat', async () => {
      const mockHandler = vi.fn();
      const { container, component } = render(CommandPalette);
      component.$on('select', mockHandler);

      const option = container.querySelector('[data-testid="palette-option-chat"]') as HTMLElement;
      await fireEvent.click(option);

      expect(mockHandler.mock.calls[0][0].detail).toBe('chat');
    });

    it('Review option emits review', async () => {
      const mockHandler = vi.fn();
      const { container, component } = render(CommandPalette);
      component.$on('select', mockHandler);

      const option = container.querySelector('[data-testid="palette-option-review"]') as HTMLElement;
      await fireEvent.click(option);

      expect(mockHandler.mock.calls[0][0].detail).toBe('review');
    });

    it('Terminal option emits terminal', async () => {
      const mockHandler = vi.fn();
      const { container, component } = render(CommandPalette);
      component.$on('select', mockHandler);

      const option = container.querySelector('[data-testid="palette-option-terminal"]') as HTMLElement;
      await fireEvent.click(option);

      expect(mockHandler.mock.calls[0][0].detail).toBe('terminal');
    });

    it('Close Panel option emits null', async () => {
      const mockHandler = vi.fn();
      const { container, component } = render(CommandPalette);
      component.$on('select', mockHandler);

      const option = container.querySelector('[data-testid="palette-option-close"]') as HTMLElement;
      await fireEvent.click(option);

      expect(mockHandler.mock.calls[0][0].detail).toBe(null);
    });
  });

  describe('selection reset', () => {
    it('resets selection to first when filter changes', async () => {
      const { container } = render(CommandPalette);
      const input = container.querySelector('input[data-testid="palette-input"]') as HTMLInputElement;

      // Move selection down
      await fireEvent.keyDown(input, { key: 'ArrowDown' });
      await fireEvent.keyDown(input, { key: 'ArrowDown' });
      await tick();

      // Type to filter
      await fireEvent.input(input, { target: { value: 'chat' } });
      await tick();

      const options = container.querySelectorAll('[data-testid^="palette-option-"]');
      expect(options[0].classList.contains('selected')).toBe(true);
    });
  });
});
