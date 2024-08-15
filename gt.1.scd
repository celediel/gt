gt(1) ["gt version v0.0.3" ["User Commands"]]

# NAME

gt \- manual page for gt version 0.0.3

# DESCRIPTION

## NAME:

gt \- Simple command line XDG Trash application

## USAGE:

gt [global options] [command [command options] / filename(s)]

## INFO:

g(o)t(rash) is a simple, command line program to interface with the XDG Trash. Files in the trash may be listed, cleaned, or restored via an interactive table, and filtered with various  flags.

## VERSION:

0.0.3

## AUTHOR:

Lilian Jónsdóttir <lilian.jonsdottir@gmail.com>

# INTERACTIVE MODE

Run with no args to start interactive mode. In interactive mode, files in the trash are displayed, and may be selected to either restore or remove permanently.

# RM-LIKE TRASHING

Run with no command and only filename(s) as argument(s) to skip displaying files, sending them straight to the trash, in a quick, rm-like way.

# COMMANDS

## TRASH:
_command_: trash, tr
	Trash a file or files

_usage_:
	trash [command options] [filename(s)]

_info_:
	The trash command finds files on the filesystem based on the filter flags and any filename args, and displays them in an interactive table, allowing them to be sorted, filtered, and selectively trashed.

_flags_:
	*--recursive*, *-r*
		operate on files recursively

	*--work-dir* dir, *-w* dir
		operate on files in this directory

	*--hidden*, *-h*
		operate on hidden files

## LIST:
_command_: list, ls
	List trashed files

_usage_:
	list [command options] [filename(s)]

_info_:
	The list command finds files in the trash based on the filter flags and any filename args, and displays them in an interactive table, allowing them to be sorted, filtered.

_flags:_
	*--non-interactive*, *-n*
		list files and quit

	*--original-path* dir, *-O* dir
		list files trashed from this directory

## RESTORE:
_command_: restore, re
	Restore a trashed file or files

_usage_:
	restore [command options] [filename(s)]

_info_:
	The restore command finds files in the trash based on the filter flags and any filename args, and displays them in an interactive table, allowing them to be sorted, filtered, and selectively restored.

_flags:_
	*--all*, *-a*
		operate on all files in trash

	*--original-path* dir, *-O* dir
		restore files trashed from this directory

## CLEAN:
_command_: clean, cl
	Clean files from trash

_usage_:
	clean [command options] [filename(s)]

_info_:
	The clean command finds files in the trash based on the filter flags and any filename args, and displays them in an interactive table, allowing them to be sorted, filtered, and selectively removed permanently.

_flags:_
	*--all*, *-a*
		operate on all files in trash

	*--original-path* dir, *-O* dir
		remove files trashed from this directory

# GLOBAL FLAGS

*--confirm*, *-c*
	ask for confirmation before executing any action

*--log* level, *-l* level
	set log level

# FILTER FLAGS (USABLE WITH ALL COMMANDS)

*--match* pattern, *-m* pattern
	operate on files matching regex pattern
	
*--glob* pattern, *-m* pattern
	operate on files matching glob
	
*--not-match* pattern, *-M* pattern
	operate on files not matching regex pattern
	
*--not-glob* pattern, *-G* pattern
	operate on files not matching glob
	
*--on* date, *-O* date
	operate on files modified on date
	
*--before* date, *-B* date
	operate on files modified before date
	
*--after* date, *-A* date
	operate on files modified after date

*--files-only*, *-F*
	operate on files only

*--dirs-only*, *-D*
	operate on directories only

*--min-size* size, *-N* size
	operate on files larger than size

*--max-size* size, *-X* size
	operate on files smaller than size

*--mode* mode, *-x* mode
	operate on files matching mode mode
