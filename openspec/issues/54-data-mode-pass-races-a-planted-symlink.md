# 54 The data mode pass can follow a symlink swapped in after find checked it

postinst's `set_data_modes` runs, as root, `find "$NEW_DATA" -type d -exec chmod u=rwx,go=rx {} +`.
`find` checks the type and `chmod` resolves the path again later, so a process running as `easee`
during configure can replace a checked directory with a symlink and get any root-owned target set
to 0755 (PR #242 review, P1). Needs an `easee` process alive during configure.

Reproduce: run postinst configure while an `easee` loop swaps a directory under `/var/lib/futurehome/easee`
for a symlink to a root-owned directory.

Close: change modes without following links (e.g. run the pass as `easee` via fh-drop, or `chmod` from
inside each directory with no-follow), then drop the limitation from packaging "Symlink Defences In postinst".
