# TODO
After completing any of these, move it to the "Change Summaries" section with a summary of the change, following the existing format.

- Support install scripts as packages (`curl -fsSL https://claude.ai/install.sh | bash`)
- Should we add a fzf-style search of local packages? ("local" means installed or bookmarked)
    - slash to search local
    - while search bar is focused, slash toggles between local and remote search
- Should we combine packages from all installed package managers in the same list? In that case, we'd want each item to have an icon or something representing the package manager.
- Add compatibility with other package managers: Arch
- Let packages.yml include more data like
    - auto-install
    - custom install instructions


# Change Summaries

#### Install function in apt (linux) seems not to work, probably a sudo problem
    
  Root cause: Bubbletea puts the terminal in raw mode and owns stdin. When sudo prompted for a password,
   the keystrokes were being intercepted by bubbletea's input handler — so sudo never received the
  actual password characters.

  Fix: Replaced the installPackage/uninstallPackage functions to use tea.ExecProcess instead of running
  the command in a background tea.Cmd closure. tea.ExecProcess suspends the TUI (restores the terminal
  to normal mode), gives the subprocess full terminal control for password input, then resumes the TUI
  when the command finishes.

  This required adding a Command(ctx, action, pkg) *exec.Cmd method to the PackageManager interface so
  the TUI layer can get the *exec.Cmd to pass to tea.ExecProcess. The existing Install/Uninstall methods
   are preserved for non-TUI use.


#### Sudo password input via TUI modal

  Previously, install/uninstall on apt used `tea.ExecProcess` which suspended the entire TUI to let sudo
  get the password from the terminal.

  Fix: Added a `NeedsSudo() bool` method to the PackageManager interface (true for apt, false for brew)
  and a `SudoCached()` helper that runs `sudo -n true` to check if credentials are already cached. When
  the user confirms an install/uninstall on a sudo manager without cached credentials, a password modal
  appears with a masked textinput. The password is piped to `sudo -S` via stdin, keeping the TUI visible
  the whole time. The password is cleared from memory immediately after use. When sudo credentials are
  already cached, the modal is skipped entirely.

#### "v" view toggle (bookmarked / manual / all)

  Replaced the "a" keybinding (which toggled between manual and all) with "v" that cycles through three
  view filters: bookmarked (only bookmarked packages), manual (bookmarked + manually installed), and all
  (every installed package). The header label updates to reflect the current filter.

#### other unsummarized changes
- brew list has a `--installed-on-request` flag, which we could use to filter out packages installed indirectly as dependencies. Does apt have this too?
- [x] After bookmarking search results and escaping back to main view, those bookmarked packages should be in the list (uninstalled of course)
- [x] Why does "Loading..." take so dang long, like 5 seconds??
- [x] The Packages list is nice and fast now, but the "Search Results" list still takes a very very long to show.
- [x] Bug: The scrolling is misbehaving. When I move the cursor up at the top edge so that the whole list should scroll, only the top 2 list items scroll, and the rest stay the same. The rest (which stayed the same) update after pushing down arrow. This only happens sometimes, which make sit even more mysterious.
- [x] Bug: When the package list (either main list or search results) has more items than can fit on the screen, the title bar ("Boxy [brew]") doesn't show 
- [x] Better loading: on search, instead of clearing the whole screen and showing just "Loading", show the search results screen, but where the list items would appear, put some loading animation
- [x] Better loading: when I hit enter to see item details, there's no indication that anything is happening until the response comes back. Instead, can we open the overlay box right away, with a loading animation until the info replaces it?
-
