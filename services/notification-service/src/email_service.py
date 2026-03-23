import boto3
from botocore.exceptions import ClientError
from src.utils.config import config
from src.utils.logger import logger

class EmailService:
    def __init__(self):
        self._client = boto3.client(
            "ses",
            region_name=config.AWS_REGION,
            aws_access_key_id=config.AWS_ACCESS_KEY_ID,
            aws_secret_access_key=config.AWS_SECRET_ACCESS_KEY
        )
        
        self._sender = config.SES_SENDER_EMAIL
        
    def send(self, to_email: str, subject: str, body: str) -> bool:
        try:
            self._client.send_email(Source=self._sender,
                                    Destination={"ToAddresses": [to_email]},
                                    Message={
                                        "subject":{"Data": subject, "Charset": "UTF-8"},
                                        "Body": {"Html": {"Data": body, "Charset": "UTF-8"}}
                                    })
            logger.info(f"Email sent to {to_email}", extra={"subject": subject})
            return True
        except ClientError as e:
            logger.error(f"Failed to send email: {e.response['Error']['Message']}", extra={
                "to": to_email,
            })
            return False
        
    def send_order_created(self, to_email: str, username: str, order_id: str, book_titles: list[str], due_date: str):
        books_html = "".join(f"<li>{t}</li>" for t in book_titles)
        body = f"""
        <h2>Order Created</h2>
        <p>Hello {username},</p>
        <p>Your order <strong>{order_id}</strong> has been created successfully.</p>
        <p>Books in your order:</p>
        <ul>{books_html}</ul>
        <p>Due Date: {due_date}</p>
        """
        self.send(to_email, "Your Order Has Been Created", body)
        
    def send_order_canceled(self, to_email: str, username: str, order_id: str):
        body = f"""
        <h2>Order Canceled</h2>
        <p>Hello {username},</p>
        <p>Your order <strong>{order_id}</strong> has been canceled.</p>
        """
        self.send(to_email, "Your Order Has Been Canceled", body)
        
    def send_order_status_updated(self, to_email: str, username: str, order_id: str, new_status: str):
        status_map = {
            "APPROVED": "approved",
            "BORROWED": "borrowed",
            "RETURNED": "returned",
            "OVERDUE": "overdue",
        }
        
        body = f"""
        <h2>Order Status Updated</h2>
        <p>Hello {username},</p>
        <p>Your order <strong>{order_id}</strong> has been {status_map.get(new_status, new_status).lower()}.</p>
        """
        self.send(to_email, "Your Order Status Has Been Updated", body)