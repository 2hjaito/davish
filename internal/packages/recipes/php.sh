case $PM in
  apt) pkg php-fpm php-cli php-mysql php-pgsql php-curl php-xml php-mbstring php-zip ;;
  dnf) pkg php-fpm php-cli php-mysqlnd php-pgsql php-xml php-mbstring; start php-fpm ;;
  apk) pkg php83 php83-fpm php83-mysqli php83-pdo_pgsql php83-curl php83-xml php83-mbstring; start php-fpm83 ;;
esac
