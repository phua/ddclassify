# . ./ddclassify.sh

_comp_ddclassify()
{
    local cur prev opts

    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="-h --help -t -a -i -d -p -r -e -c -m -misfile -x -depth -g -v"
    sopt="-x -depth -g -v"

    case "${prev}" in
        -d|-c|-e)
            COMPREPLY=( $(compgen -d ${cur} ))
            return
            ;;
        -m)
            COMPREPLY=( $(compgen -W "{1,2,4,8,17,18,20,24}" -- ${cur}) )
            return
            ;;
        -x)
            _filedir 'xml'
            return
            ;;
        -depth)
            COMPREPLY=( $(compgen -W "{1..3}" -- ${cur}) )
            return
            ;;
        *)
            ;;
    esac

    # for (( j=${#COMP_WORDS[@]}-1 ; j>=0 ; j-- )); do
    #     i="${COMP_WORDS[j]}"
    for i in "${COMP_WORDS[@]}"; do
        if [[ "$i" == "-t" ]]; then
            COMPREPLY=( $(compgen -W "-a ${sopt}" -- ${cur}) )
            return
        elif [[ "$i" == "-a" ]]; then
            COMPREPLY=( $(compgen -W "-t ${sopt}" -- ${cur}) )
            return
        elif [[ "$i" == "-i" ]]; then
            COMPREPLY=( $(compgen -W "${sopt}" -- ${cur}) )
            return
        elif [[ "$i" == "-d" ]]; then
            COMPREPLY=( $(compgen -W "-p -r -e -c -m -misfile ${sopt}" -- ${cur}) )
            return
        fi
    done

    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
}

complete -F _comp_ddclassify ddclassify
