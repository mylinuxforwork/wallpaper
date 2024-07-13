# ------------------------------------------------------
# Install wallpapers
# ------------------------------------------------------
clear
echo -e "${GREEN}"
cat <<"EOF"
__        __    _ _                             
\ \      / /_ _| | |_ __   __ _ _ __   ___ _ __ 
 \ \ /\ / / _` | | | '_ \ / _` | '_ \ / _ \ '__|
  \ V  V / (_| | | | |_) | (_| | |_) |  __/ |   
   \_/\_/ \__,_|_|_| .__/ \__,_| .__/ \___|_|   
                   |_|         |_|              

EOF
echo -e "${NONE}"
if [ ! -d .git ]; then
    echo "Do you want to download the wallpapers from repository "
    echo "https://github.com/mylinuxforwork/wallpaper/ ?"
    echo ""
    if gum confirm "Do you want to download the repository?" ;then
        if [ ! -d ~/Downloads/wallpaper ]; then
            mkdir -p ~/Downloads/wallpaper
        fi
        wget -P ~/Downloads/wallpaper https://github.com/mylinuxforwork/wallpaper/archive/refs/heads/main.zip
        unzip -o ~/Downloads/wallpaper/main.zip -d ~/Downloads/wallpaper/
        if [ ! -d ~/wallpaper/ ]; then
            mkdir ~/wallpaper
        fi
        cp ~/Downloads/wallpaper/* ~/wallpaper/
        echo "Wallpapers from the repository installed successfully."
    elif [ $? -eq 130 ]; then
        exit 130
    else
        exit
    fi
else
    echo "Wallpapers downloaded with git. Update with git pull."
    if gum confirm "Do you want to pull updates from the repository?" ;then
        git pull
        echo "Wallpapers updated successfully from the repository."
    elif [ $? -eq 130 ]; then
        exit 130
    else
        exit
    fi
fi
echo ""
