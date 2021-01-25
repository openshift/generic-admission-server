FROM fedora:33

RUN dnf install -y glibc-devel \
                                	&& dnf clean all \
                                	&& rm -rf /var/cache/yum

ADD dump-request-admission-webhook /usr/bin
