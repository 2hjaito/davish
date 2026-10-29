case $PM in
  apt) pkg postgresql ;;
  dnf) pkg postgresql-server; sudo postgresql-setup --initdb ;;
  apk) pkg postgresql ;;
esac
start postgresql
