# fish completion for gt                                  -*- shell-script -*-

set -l commands list ls trash tr clean cl restore re
set -l already_in_trash_commands list ls clean cl restore re
set -l trash_commands trash tr
set -l log_levels debug info warn error fatal

# commands
complete -c gt -f -n "not __fish_seen_subcommand_from $commands" -a "list ls" -d "list trashed files"
complete -c gt -F -n "not __fish_seen_subcommand_from $commands" -a "trash tr" -d "trash a file or files"
complete -c gt -f -n "not __fish_seen_subcommand_from $commands" -a "restore re" -d "restore files from trash"
complete -c gt -f -n "not __fish_seen_subcommand_from $commands" -a "clean cl" -d "clean files from trash"

# flags
complete -c gt -n "not __fish_seen_subcommand_from $commands" -l help -s h -d "show help"
complete -c gt -n "not __fish_seen_subcommand_from $commands" -l version -s v -d "show version"
complete -c gt -n "not __fish_seen_subcommand_from $commands" -l log -s l -d "log level" -fra (string join " " $log_levels)

# everyone flags
complete -c gt -rf -n "__fish_seen_subcommand_from $commands" -l match -s m -d "operate on files matching regex pattern"
complete -c gt -rf -n "__fish_seen_subcommand_from $commands" -l not-match -s M -d "operate on files not matching regex pattern"
complete -c gt -rf -n "__fish_seen_subcommand_from $commands" -l glob -s g -d "operate on files matching glob pattern"
complete -c gt -rf -n "__fish_seen_subcommand_from $commands" -l not-glob -s G -d "operate on files not matching glob pattern"
complete -c gt -rf -n "__fish_seen_subcommand_from $commands" -l files-only -s F -d "operate on files only"
complete -c gt -rf -n "__fish_seen_subcommand_from $commands" -l dirs-only -s D -d "operate on dirs only"
complete -c gt -rf -n "__fish_seen_subcommand_from $commands" -l hidden -s H -d "operate on hidden files"
complete -c gt -rf -n "__fish_seen_subcommand_from $commands" -l before -s b -d "operate on files before date"
complete -c gt -rf -n "__fish_seen_subcommand_from $commands" -l after -s a -d "operate on files after date"
complete -c gt -rf -n "__fish_seen_subcommand_from $commands" -l on -s o -d "operate on files on date"
complete -c gt -rf -n "__fish_seen_subcommand_from $commands" -l min-size -s N -d "operate on files larger than size"
complete -c gt -rf -n "__fish_seen_subcommand_from $commands" -l max-size -s X -d "operate on files smaller than size"
complete -c gt -rf -n "__fish_seen_subcommand_from $commands" -l mode -s x -d "operate on files matching mode"

# trash flags
complete -c gt -rf -n "__fish_seen_subcommand_from $trash_commands" -l recursive -s r -d "recursively trash files"
complete -c gt -rf -n "__fish_seen_subcommand_from $trash_commands" -l work-dir -s w -d "trash files in specified directory"

# list / clean / restore flags
complete -c gt -rf -n "__fish_seen_subcommand_from $already_in_trash_commands" -l original-path -s O -d "operate on files trashed from this directory"
