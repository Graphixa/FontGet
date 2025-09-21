## SEARCH

- The search command seems to update the sources from online sources every time run. Confirm that this is not true. The sources should be updated daily (>24 hrs apart from last run) or can be updated manually using fontget sources update.


## SOURCES

- Fontget Sources Manage command needs fixing. needs to look similar to ther commands with table layout for easier interpetation of the list of items. Add a header to the list.
- Bubbletea has a component for a table which may be worth investigating.
- Need to remove fontget sources "list" subcommand as this is better handled in the manage subcommand.
- Need to fixup the fontget sources "info" subcommand as it doesn't contain much info that is specificly not covered in the sources manage subcommand. Needs to be useful otherwise it get's clipepd.
  - Maybe could include things like last updated sources date. Source file location.

