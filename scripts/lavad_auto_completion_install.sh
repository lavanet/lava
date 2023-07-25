#!/bin/bash

function get_terminal_type() {
  # Check if the SHELL environment variable is set
  if [ -n "$SHELL" ]; then
    # Extract the name of the shell from the path
    shell_name=$(basename "$SHELL")
    # Check the name of the shell and return the type
    if [ "$shell_name" = "bash" ]; then
	echo "bash"
    elif [ -f "$HOME/.oh-my-zsh/oh-my-zsh.sh" ]; then
      echo "oh-my-zsh"
    elif [ "$shell_name" = "zsh" ]; then
      echo "zsh"
    else
      echo "unknown"
    fi
  else
    echo "unknown"
  fi
}

# get terminal type to pick the right command
# check that lavad is installed
terminal_type=$(get_terminal_type)
case $terminal_type in
	bash)
		# create the completion script
		lavad completion > /tmp/_lavad
		comp_dir="$HOME/.bash_completion.d"
		# move the completion script to /etc/bash_completion.d/lavad
		if [ ! -d $comp_dir ]; then
			mkdir $comp_dir
		fi
		mv /tmp/_lavad $comp_dir
		echo " * Created an auto-completion script in $comp_dir"

		bashrc_file="$HOME/.bashrc"
		if [ -f "$HOME/.bash_profile" ]; then
			bashrc_file="$HOME/.bash_profile"
		fi

		if ! grep -q "source $file" $bashrc_file; then
			echo >> $bashrc_file
			echo "if [ -d $comp_dir ]; then" >> $bashrc_file
			echo "	for file in $comp_dir/*; do"  >> $bashrc_file
			echo "		source \$file" >> $bashrc_file
			echo "	done" >> $bashrc_file
			echo "fi" >> $bashrc_file
		fi

		echo " * Sourced $comp_dir/_lavad"

		# create the completion script
		lava-protocol completion > /tmp/_lava-protocol
		comp_dir="$HOME/.bash_completion.d"
		# move the completion script to /etc/bash_completion.d/lava-protocol
		if [ ! -d $comp_dir ]; then
			mkdir $comp_dir
		fi
		mv /tmp/_lava-protocol $comp_dir
		echo " * Created an auto-completion script in $comp_dir"

		bashrc_file="$HOME/.bashrc"
		if [ -f "$HOME/.bash_profile" ]; then
			bashrc_file="$HOME/.bash_profile"
		fi

		if ! grep -q "source $file" $bashrc_file; then
			echo >> $bashrc_file
			echo "if [ -d $comp_dir ]; then" >> $bashrc_file
			echo "	for file in $comp_dir/*; do"  >> $bashrc_file
			echo "		source \$file" >> $bashrc_file
			echo "	done" >> $bashrc_file
			echo "fi" >> $bashrc_file
		fi

		echo " * Sourced $comp_dir/_lava-protocol"
		;;
	zsh)
                # create the completion script
		lavad completion --zsh > /tmp/_lavad
		comp_dir="$HOME/.zsh/completion"

                # move the completion script to $HOME/.zsh/completion
                if [ ! -d $comp_dir ]; then
                        mkdir -p $comp_dir
                fi
                mv /tmp/_lavad $comp_dir
		echo " * Created an auto-completion script in $comp_dir"

		# add necessary lines to $HOME/.zshrc
                echo 'fpath=(~/.zsh/completion $fpath)' >> ~/.zshenv

		lava-protocol completion --zsh > /tmp/_lava-protocol
		comp_dir="$HOME/.zsh/completion"

                # move the completion script to $HOME/.zsh/completion
                if [ ! -d $comp_dir ]; then
                        mkdir -p $comp_dir
                fi
                mv /tmp/_lava-protocol $comp_dir
		echo " * Created an auto-completion script in $comp_dir"

		# add necessary lines to $HOME/.zshrc
                echo 'fpath=(~/.zsh/completion $fpath)' >> ~/.zshenv
		;;
	oh-my-zsh)
		# create the completion script
		lavad completion --zsh > /tmp/_lavad

                # move the completion script to $HOME/.oh-my-zsh/custom/plugins/lavad
                if [ ! -d "$HOME/.oh-my-zsh/custom/plugins/lavad" ]; then
                        mkdir $HOME/.oh-my-zsh/custom/plugins/lavad
                fi
                mv /tmp/_lavad $HOME/.oh-my-zsh/custom/plugins/lavad
		echo " * Created an auto-completion script in $HOME/.oh-my-zsh/custom/plugins/lavad"

		# edit the oh-my-zsh plugins defined in $HOME/.zshrc
		plugins_line=$(grep -n "^plugins=(" $HOME/.zshrc | cut -d ":" -f 1)
		plugins=$(sed -n "${plugins_line}p" $HOME/.zshrc | cut -d "(" -f 2 | cut -d ")" -f 1)
		new_plugins="lavad ${plugins}"
		sed -i "${plugins_line}s/${plugins}/${new_plugins}/" $HOME/.zshrc

		# apply your .zshrc changes
                source ~/.zshrc 2> /dev/null
		echo " * Updated .zshrc and sourced"
		;;
	*)
		if [[ $? -eq 2 ]]; then
			echo "For bash terminal, the script requires sudo privileges"
			exit 2
		fi
		echo "Unknown terminal type"
		exit 1
		;;
esac

echo " ** Completion script installed !"
echo " ** Please open a new terminal to start using lavad with auto-completion :)"
