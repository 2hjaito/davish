case $PM in
  apt) pkg redis-server; start redis-server ;;
  *) pkg redis; start redis ;;
esac
