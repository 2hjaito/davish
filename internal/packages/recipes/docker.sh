case $PM in
  apt) pkg docker.io ;;
  dnf) pkg moby-engine docker-compose ;;
  apk) pkg docker docker-cli-compose ;;
esac
start docker
# Lets you run docker without sudo after you log in again.
sudo usermod -aG docker "$USER"
