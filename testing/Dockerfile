FROM python:3.11-slim-buster

ENV VIRTUAL_ENV=/opt/venv
ENV PYTHONUNBUFFERED=1
RUN python3 -m venv  ${VIRTUAL_ENV}
ENV PATH="${VIRTUAL_ENV}/bin:$PATH"

COPY requirements.txt . 

RUN pip install --upgrade pip && \
    apt-get update && \
    pip install --no-cache-dir -r requirements.txt

COPY . . 

CMD ["python", "/app/main.py"]