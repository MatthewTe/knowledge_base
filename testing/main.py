import fastapi
from fastapi.responses import Response
from fastapi.middleware.cors import CORSMiddleware
import uvicorn
import os

app = fastapi.FastAPI()

origins = [
    'http://localhost:8080'
]

app.add_middleware(
    CORSMiddleware,
    allow_origins=origins,
    allow_credentials=True,
    allow_methods=["GET"],
    allow_headers=["*"],
)

@app.get("/test/rss_feed")
async def return_demo_xml_page():
    "Returning an XML page of an RSS Feed for testing"
    xml_path: str = "./38_north_test.rss"
    
    if not os.path.exists(xml_path):
        return Response(content="XML File not found", status_code=404)
    
    with open(xml_path, "rb") as f:
        xml_data = f.read()

    response = Response(content=xml_data, media_type="application/xml")
    return response

if __name__ == '__main__':
    uvicorn.run(app, host='0.0.0.0', port=8000)