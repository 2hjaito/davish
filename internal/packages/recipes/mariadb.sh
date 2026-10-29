case $PM in
  apk) pkg mariadb mariadb-client; sudo mariadb-install-db --user=mysql --datadir=/var/lib/mysql ;;
  *) pkg mariadb-server ;;
esac
start mariadb
