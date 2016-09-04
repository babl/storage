FROM busybox
ADD babl-storage_linux_amd64 /bin/babl-storage
CMD ["/bin/babl-storage"]
