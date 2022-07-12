## **dlog** - docker logs viewer

Построен на базе программы Slit и использует теже самые терминальные возможности:

### Key Bindings:

##### Search/Filters
- `/` - Forward search
- `?` - Backsearch
- `n` - Next match
- `N` - Previous match
- `CTRL + /` - Switch between `CaseSensitive` search and `RegEx`
- `&` - Filter: intersect
- `-` - Filter: exclude
- `+` - Filter: union
- `=` - Remove all filters
- `U` - Removes last filter
- `C` - Stands for "Context", switches off/on all filters, helpful to get context of current line (which is the first line, at the top of the screen)

##### Navigation
- `f`, `PageDown`, `Space`, `CTRL + F` - Page Down
- `CTRL + D` - Half page down
- `b`, `PageUp`, `CTRL + B` - Page Up
- `CTRL + U` - Half page up
- `g`, `Home` - Go to first line
- `G`, `End` - Go to last line
- `Arrow down`, `j` - Move one line down
- `Arrow up`, `k` - Move one line up
- `Arrow left`, `Arrow right` - Scroll between docker containers
- `<`, `>` - Precise horizontal scrolling, 1 character a time

##### Misc
- `K` - Keep N first characters(usually containing timestamp) when navigating horizontally
  Up/Down arrows during K-mode will adjust N of kept chars
- `W` - Wrap/Unwrap lines
- `q`, `ESC` - quit

### Search Modes
Both search and filters currently support the `CaseSensitive` and `RegEx` modes.
To switch between modes press `CTRL + /` in search/filter input.

### Highlighting
- ``` ` ``` - (Backtick) Mark top line for highlighting (i.e will be shown no matter what are other filters)
- ``` ~ ``` - Highlight filter. I.e search and highlight everything that matches
- `h` - Move to next highlighted line
- `H` - Move to previous highlighted line
- `ctrl+h` - Remove all highlights
- `=` - Removes only filters, does not remove highlights via `~`
- 