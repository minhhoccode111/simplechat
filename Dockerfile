FROM scratch
COPY simplechat /simplechat
COPY index.html /index.html
ENV TZ=Asia/Ho_Chi_Minh
ENTRYPOINT ["/simplechat"]
