#!/usr/bin/env bash

# ----------------------------------------------------------------------

# loader.bash
#
# This script implements Shell Script Loader for all versions of Bash
# starting 2.04.
#
# The script works faster with associative arrays.  To use associative
# arrays, run the script with Bash 4.2 or newer.  You can also enable
# usage of associative arrays with 4.0 or 4.1 by including it globally;
# that is, not include the script with 'source' or '.' inside any
# function.
#
# Please see loader.txt for more info on how to use this script.
#
# This script complies with the Requiring Specifications of
# Shell Script Loader version 0 (RS0).
#
# Version: 0.2.2
#
# Author: konsolebox
# Copyright Free / Public Domain
# Aug. 29, 2009 (Last Updated 2018/01/29)

# Limitations of Shell Script Loader with integers and associative
# arrays:
#
# With versions of Bash earlier than 4.2, a variable can't be declared
# global with the use of 'typeset' and 'declare' builtins when inside a
# function.  With Shell Script Loader, shell scripts are always loaded
# inside functions so variables that can only be declared using the said
# builtin commands cannot be declared global.  These kinds of variables
# that cannot be declared global are the newer types like associative
# arrays and integers.  Unlike Zsh, we can add '-g' as an option to
# 'typeset' or 'declare' to declare global variables but we can't do
# that in Bash.
#
# For example, if we do something like
#
# > include file.sh
#
# Where the contents of file.sh is
#
# > declare -A ASSOCIATIVE_ARRAY
# > declare -i INTEGER
#
# After include() ends, the variables automatically gets lost since
# variables are only local and not global if declare or typeset is used
# inside a function and we know that include() is a function.
#
# However it's safe to declare other types of variables like indexed
# arrays in simpler way.
#
# For example:
#
# > SIMPLE_VAR=''
# > ARRAY_VAR=()
#
# These declarations are even just optional.
#
# Note: These conditions do not apply if you only plan to run the code
# in compiled form since you no longer have to use the functions.  For
# more info about compilation, please see the available compilers of
# Shell Script Loader.

# ----------------------------------------------------------------------

if [ "$LOADER_ACTIVE" = true ]; then
	echo 'loader: Loader cannot be loaded twice.' >&2
	exit 1
fi

if [ -z "$BASH_VERSION" ]; then
	echo 'loader: Bash is needed to run this script.' >&2
	exit 1
fi

if ! [ "$BASH_VERSINFO" -ge 3 -o "$BASH_VERSION" '>' 2.03 ]; then
	echo 'loader: This script is only compatible with versions of Bash not earlier than 2.04.' >&2
	exit 1
fi

if ! declare -a LOADER_TEST_0; then
	echo 'loader: This build of Bash does not support arrays.' >&2
	exit 1
fi

#### PUBLIC VARIABLES ####

LOADER_ACTIVE=true
LOADER_RS=0
LOADER_VERSION=0.2.2

#### PRIVATE VARIABLES ####

LOADER_CS=()
LOADER_CS_I=0
LOADER_PATHS=()

if [[ BASH_VERSINFO -ge 5 || (BASH_VERSINFO -eq 4 && BASH_VERSINFO[1] -ge 2) ]]; then
	declare -g -A LOADER_FLAGS=()
	declare -g -A LOADER_PATHS_FLAGS=()
	LOADER_USE_ASSOC_ARRAYS=true
elif [[ BASH_VERSINFO -eq 4 ]] && declare -A LOADER_TEST_1 &>/dev/null && ! local LOADER_TEST_2 &>/dev/null; then
	declare -A LOADER_FLAGS=()
	declare -A LOADER_PATHS_FLAGS=()
	LOADER_USE_ASSOC_ARRAYS=true
else
	LOADER_USE_ASSOC_ARRAYS=false
fi

#### PUBLIC FUNCTIONS ####

function load {
	[[ $# -eq 0 ]] && loader_fail "Function called with no argument." load

	case $1 in
	'')
		loader_fail "File expression cannot be null." load "$@"
		;;
	/*|./*|../*)
		if [[ -f $1 ]]; then
			loader_getcleanpath "$1"
			[[ -r $__ ]] || loader_fail "File not readable: $__" load "$@"
			loader_load "${@:2}"
			__=$?
			unset 'LOADER_CS[LOADER_CS_I--]'
			return "$__"
		fi
		;;
	*)
		for __ in "${LOADER_PATHS[@]}"; do
			[[ -f $__/$1 ]] || continue
			loader_getcleanpath "$__/$1"
			[[ -r $__ ]] || loader_fail "Found file not readable: $__" load "$@"
			loader_flag_ "$1"
			loader_load "${@:2}"
			__=$?
			unset 'LOADER_CS[LOADER_CS_I--]'
			return "$__"
		done
		;;
	esac

	loader_fail "File not found: $1" load "$@"
}

function include {
	[[ $# -eq 0 ]] && loader_fail "Function called with no argument." include

	case $1 in
	'')
		loader_fail "File expression cannot be null." include "$@"
		;;
	/*|./*|../*)
		loader_getcleanpath "$1"
		loader_flagged "$__" && return

		if [[ -f $__ ]]; then
			[[ -r $__ ]] || loader_fail "File not readable: $__" include "$@"
			loader_load "${@:2}"
			__=$?
			unset 'LOADER_CS[LOADER_CS_I--]'
			return "$__"
		fi
		;;
	*)
		loader_flagged "$1" && return

		for __ in "${LOADER_PATHS[@]}"; do
			loader_getcleanpath "$__/$1"

			if loader_flagged "$__"; then
				loader_flag_ "$1"
				return 0
			elif [[ -f $__ ]]; then
				[[ -r $__ ]] || loader_fail "Found file not readable: $__" include "$@"
				loader_flag_ "$1"
				loader_load "${@:2}"
				__=$?
				unset 'LOADER_CS[LOADER_CS_I--]'
				return "$__"
			fi
		done
		;;
	esac

	loader_fail "File not found: $1" include "$@"
}

function call {
	[[ $# -eq 0 ]] && loader_fail "Function called with no argument." call

	case $1 in
	'')
		loader_fail "File expression cannot be null." call "$@"
		;;
	/*|./*|../*)
		if [[ -f $1 ]]; then
			loader_getcleanpath "$1"
			[[ -r $__ ]] || loader_fail "File not readable: $__" call "$@"
			( loader_load "${@:2}" )
			return
		fi
		;;
	*)
		for __ in "${LOADER_PATHS[@]}"; do
			[[ -f $__/$1 ]] || continue
			loader_getcleanpath "$__/$1"
			[[ -r $__ ]] || loader_fail "Found file not readable: $__" call "$@"

			(
				loader_flag_ "$1"
				loader_load "${@:2}"
			)

			return
		done
		;;
	esac

	loader_fail "File not found: $1" call "$@"
}

function loader_addpath {
	for __; do
		[[ -d $__ ]] || loader_fail "Directory not found: $__" loader_addpath "$@"
		[[ -x $__ ]] || loader_fail "Directory not accessible: $__" loader_addpath "$@"
		[[ -r $__ ]] || loader_fail "Directory not searchable: $__" loader_addpath "$@"
		loader_getcleanpath "$__"
		loader_addpath_ "$__"
	done
}

function loader_flag {
	[[ $# -eq 1 ]] || loader_fail "Function requires a single argument." loader_flag "$@"
	loader_getcleanpath "$1"
	loader_flag_ "$__"
}

function loader_reset {
	if [[ $# -eq 0 ]]; then
		loader_reset_flags
		loader_reset_paths
	elif [[ $1 == flags ]]; then
		loader_reset_flags
	elif [[ $1 == paths ]]; then
		loader_reset_paths
	else
		loader_fail "Invalid argument: $1" loader_reset "$@"
	fi
}

function loader_finish {
	LOADER_ACTIVE=false
	loader_reset_flags

	unset -v LOADER_CS LOADER_CS_I LOADER_FLAGS LOADER_PATHS \
			LOADER_PATHS_FLAGS

	unset -f load include call loader_addpath loader_addpath_ \
			loader_fail loader_finish loader_flagged loader_flag \
			loader_flag_ loader_getcleanpath loader_load loader_load \
			loader_reset_flags loader_reset_paths loader_reset
}

#### PRIVATE FUNCTIONS ####

function loader_load {
	loader_flag_ "$__"
	LOADER_CS[++LOADER_CS_I]=$__
	. "$__"
}

function loader_fail {
	local message=$1 func=$2 main='(main)'
	[[ -n $0 && "${0##*/}" != "${BASH##*/}" ]] && main=$0
	shift 2

	{
		echo "loader: $func(): $message"
		echo
		echo '  Current scope:'

		if [[ LOADER_CS_I -gt 0 ]]; then
			echo "    ${LOADER_CS[LOADER_CS_I]}"
		else
			echo "    $main"
		fi

		echo

		if [[ $# -gt 0 ]]; then
			echo '  Command:'
			echo -n "    $func"
			printf ' %q' "$@"
			echo
			echo
		fi

		if [[ LOADER_CS_I -gt 0 ]]; then
			echo '  Call stack:'
			echo "    $main"
			printf '    -> %s\n' "${LOADER_CS[@]}"
			echo
		fi

		echo '  Search paths:'

		if [[ ${#LOADER_PATHS[@]} -gt 0 ]]; then
			printf '    %s\n' "${LOADER_PATHS[@]}"
		else
			echo '    (empty)'
		fi

		echo
		echo '  Working directory:'
		echo "    $PWD"
		echo
	} >&2

	exit 1
}

#### VERSION DEPENDENT FUNCTIONS AND VARIABLES ####

if [[ $LOADER_USE_ASSOC_ARRAYS == true ]]; then
	function loader_addpath_ {
		if [[ -z ${LOADER_PATHS_FLAGS[$1]} ]]; then
			LOADER_PATHS[${#LOADER_PATHS[@]}]=$1
			LOADER_PATHS_FLAGS[$1]=.
		fi
	}

	function loader_flag_ {
		LOADER_FLAGS[$1]=.
	}

	function loader_flagged {
		[[ -n ${LOADER_FLAGS[$1]} ]]
	}

	function loader_reset_flags {
		LOADER_FLAGS=()
	}

	function loader_reset_paths {
		LOADER_PATHS=()
		LOADER_PATHS_FLAGS=()
	}
else
	function loader_addpath_ {
		for __ in "${LOADER_PATHS[@]}"; do
			[[ $1 == "$__" ]] && return
		done

		LOADER_PATHS[${#LOADER_PATHS[@]}]=$1
	}

	function loader_flag_ {
		local v
		v=${1//./_dt_}
		v=${v// /_sp_}
		v=${v//\//_sl_}
		v=LOADER_FLAGS_${v//[![:alnum:]_]/_ot_}
		eval "$v=."
	}

	function loader_flagged {
		local v
		v=${1//./_dt_}
		v=${v// /_sp_}
		v=${v//\//_sl_}
		v=LOADER_FLAGS_${v//[![:alnum:]_]/_ot_}
		[[ -n ${!v} ]]
	}

	function loader_reset_flags {
		unset -v "${!LOADER_FLAGS_@}"
	}

	function loader_reset_paths {
		LOADER_PATHS=()
	}
fi

function loader_getcleanpath {
	local t=() i=0 IFS=/

	case $1 in
	/*)
		__=${1#/}
		;;
	*)
		__=${PWD#/}/$1
		;;
	esac

	case $- in
	*f*)
		set -- $__
		;;
	*)
		set -f
		set -- $__
		set +f
		;;
	esac

	for __; do
		case $__ in
		..)
			(( i )) && unset 't[--i]'
			continue
			;;
		.|'')
			continue
			;;
		esac

		t[i++]=$__
	done

	__="/${t[*]}"
}

unset -v LOADER_TEST_0 LOADER_TEST_1 LOADER_TEST_2 LOADER_USE_ASSOC_ARRAYS

# ----------------------------------------------------------------------

# * Using 'set -- $VAR' to split strings inside variables will sometimes
#   yield different strings if one of the strings contain globs
#   characters like *, ? and the brackets [ and ] that are also valid
#   characters in filenames.

# * Using 'read -a' to split strings to arrays yields elements
#   that contain invalid characters when a null token is found.
#   (Bash versions < 3.0)

# * There's an odd behavior in Bash 4.3 where unsetting variables along
#   with functions does not unset the variables.

# ----------------------------------------------------------------------
