# Use Bubbletea for TUI framework

We will use Bubbletea (Charm's TUI framework) for building jdash's terminal user interface, along with its ecosystem libraries (Lipgloss for styling, Bubbles for components).

## Considered Options

**A) Bubbletea** (Elm architecture, most popular)
- **Selected**: Proven for dashboard UIs, massive ecosystem, great docs

**B) tview** (widget-based, higher-level)
- Rejected: Faster for simple forms but less flexible for custom layouts like gh-dash's split-pane design

**C) tcell** (low-level terminal control)
- Rejected: Too much work for common UI patterns, would be reinventing what Bubbletea already provides

## Why Bubbletea

**Proven for this exact use case:** gh-dash (our inspiration) uses Bubbletea successfully for a similar dashboard interface with sections, navigation, and preview panes.

**Production adoption:** Microsoft Azure, AWS, NVIDIA, Cockroach Labs, MinIO, Ubuntu, and 18,000+ other applications demonstrate it's production-ready and maintained.

**Ecosystem:** Lipgloss (styling), Bubbles (text inputs, tables, spinners, viewports), Huh (forms) provide pre-built components that match our needs.

**Architecture:** The Elm pattern (model-update-view) fits our needs — model holds issue data and UI state, update handles key presses and API responses, view renders sections/tables.

**Documentation and examples:** Extensive examples, active community, well-documented APIs make development faster.

## Consequences

- ✅ Can reference gh-dash's source code for patterns (section navigation, preview panes, keybindings)
- ✅ Bubbles provides table component for issue lists
- ✅ Lipgloss handles status coloring, borders, layout
- ✅ Strong typing and compile-time safety (vs text-based approaches)
- ⚠️ Learning curve for Elm architecture if unfamiliar
- ⚠️ Dependency on Charm ecosystem (low risk given widespread adoption)

## Implementation Dependencies

```
charm.land/bubbletea/v2
charm.land/bubbles/v2  
charm.land/lipgloss/v2
```
