# gt

g(o)t(rash) is a simple, command line program to interface with the XDG Trash. Files in the trash may be listed, cleaned, or restored via an interactive table, and filtered with various  flags.

## Interactive Mode

Run with no args to start interactive mode. In interactive mode, files in the trash are displayed, and may be selected to either restore or remove permanently.

## rm-like Trashing

Run with no command and only filename(s) as argument(s) to skip displaying files, sending them straight to the trash, in a quick, rm-like way.

## Commands

Files are displayed in an interactive table, allowing them to be sorted, filtered, and selectively operated on.

### trash / tr
Find files on the filesystem based on the filter flags and any filename args.

### list / ls
Find files in the trash based on the filter flags and any filename args.

### restore / re
Find files in the trash based on the filter flags and any filename args.

### clean / cl
Find files in the trash based on the filter flags and any filename args.

See gt(1) or `gt --help` for more in depth information on all command line flags.
