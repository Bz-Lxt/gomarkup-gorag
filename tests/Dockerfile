FROM python:3.12-alpine
ENV TZ=Asia/Shanghai PYTHONIOENCODING=utf-8
RUN pip install --no-cache-dir pytest==8.3.4 requests==2.32.3 -i https://pypi.tuna.tsinghua.edu.cn/simple
WORKDIR /tests
COPY . /tests
CMD ["pytest", "-p", "no:cacheprovider", "api_smoke.py"]
