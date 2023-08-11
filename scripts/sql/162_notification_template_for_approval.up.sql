ALTER TABLE ONLY public.notifier_event_log DROP CONSTRAINT notifier_event_log_event_type_id_fkey;
ALTER TABLE ONLY public.notifier_event_log
    ADD CONSTRAINT notifier_event_log_event_type_id_fkey FOREIGN KEY (event_type_id) REFERENCES public.event(id) ;
INSERT INTO public.event (id, event_type, description) VALUES (4, 'APPROVAL', '');
INSERT INTO "public"."notification_templates" (channel_type, node_type, event_type_id, template_name, template_payload)
VALUES ('smtp', 'CD', 4, 'CD approval smtp template', '{"from": "{{fromEmail}}",
"to": "{{toEmail}}","subject": "🛎️ Image approval requested | Application > {{appName}} | Environment > {{envName}}","html": "<table style=\"width: 600px; height: 485px;  border-collapse: collapse; padding: 20px;\"><tr style=\"background-color:#E5F2FF;\"><td colspan=\"2\" style=\"padding-left:16px;\"><h2 style=\"color:#000A14;\">Image approval request</h2><span>{{eventTime}}</span><br><span>by <strong style=\"color:#0066CC;\">{{triggeredBy}}</strong></span><br><br>{{#imageApprovalLink}}<a href=\"{{&imageApprovalLink}}\" style=\" height: 32px; padding: 7px 12px; line-height: 32px; font-size: 12px; font-weight: 600; border-radius: 4px; text-decoration: none; outline: none; min-width: 64px; text-transform: capitalize; text-align: center; background: #0066CC; color: #fff; border: 1px solid transparent; cursor: pointer;\">View request</a><br><br>{{/imageApprovalLink}}</td><td style=\"text-align: right;\"><img src=\"https://cdn.devtron.ai/images/img_build_notification.png\" style=\"height: 72px; width: 72px;\"></td></tr><tr><td colspan=\"3\"><hr><br><span>Application: <strong>{{appName}}</strong></span>&nbsp;&nbsp;|&nbsp;&nbsp;<span>Environment: <strong>{{envName}}</strong></span><br></span><br><br><hr><h3>Image Details</h3><span>Image tag <br><strong>{{imageTag}}</strong></span></td></tr><tr><td colspan=\"3\"><br><br><span style=\"display: {{commentDisplayStyle}}\">Comment<br><strong>{{comment}}</strong></span></td></tr><tr><td colspan=\"3\"><br><br><span style=\"display: {{tagDisplayStyle}}\">Tags<br><strong>{{tags}}</strong></span><br></td></tr></table>"}');
INSERT INTO "public"."notification_templates" (channel_type, node_type, event_type_id, template_name, template_payload)
VALUES ('ses', 'CD', 4, 'CD approval ses template', '{"from": "{{fromEmail}}",
"to": "{{toEmail}}","subject": "🛎️ Image approval requested | Application > {{appName}} | Environment > {{envName}}","html": "<table style=\"width: 600px; height: 485px;  border-collapse: collapse; padding: 20px;\"><tr style=\"height: 60px;\"><td><img src=\"https://cdn.devtron.ai/images/devtron-logo-horizontal-dual.png\" style=\"text-align: left; padding: 16px;\"/></td></tr><tr style=\"background-color:#E5F2FF; border-radius: 8px;\"><td colspan=\"2\" style=\"padding-left:16px; border-radius: 8px 0 0 8px;\"><h2 style=\"color:#000A14;\">Image approval request</h2><span>{{eventTime}}</span><br><span>by <strong style=\"color:#0066CC;\">{{triggeredBy}}</strong></span><br><br>{{#imageApprovalLink}}<a href=\"{{&imageApprovalLink}}\" style=\" height: 32px; padding: 7px 12px; line-height: 32px; font-size: 12px; font-weight: 600; border-radius: 4px; text-decoration: none; outline: none; min-width: 64px; text-transform: capitalize; text-align: center; background: #0066CC; color: #fff; border: 1px solid transparent; cursor: pointer;\">View request</a><br><br>{{/imageApprovalLink}}</td><td style=\"text-align: right; border-radius: 0 8px 8px 0;\"><img src=\"https://cdn.devtron.ai/images/img_build_notification.png\" style=\"height: 72px; width: 72px;\"></td></tr><tr><td colspan=\"3\"><br><span>Application: <strong>{{appName}}</strong></span>&nbsp;&nbsp;|&nbsp;&nbsp;<span>Environment: <strong>{{envName}}</strong></span><br></span><br><hr><h3>Image Details</h3><span>Image tag <br><strong>{{imageTag}}</strong></span></td></tr><tr><td colspan=\"3\"><br><br><span style=\"display: {{commentDisplayStyle}}\">Comment<br><strong>{{comment}}</strong></span></td></tr><tr><td colspan=\"3\"><br><br><span style=\"display: {{tagDisplayStyle}}\">Tags<br><strong>{{tags}}</strong></span></td></tr><br><br><hr><div style=\"display: flex;\"><div style=\"display: flex\"><span><a href=\"https://twitter.com/DevtronL?t=pfEle-aa89P_i8zV1t340w&s=09\"><img src=\"https://cdn.devtron.ai/images/twitter_social.png\"/></a></span>&nbsp;&nbsp;&nbsp;<span><a href=\"https://www.linkedin.com/company/devtron-labs/\"><img src=\"https://cdn.devtron.ai/images/linkedin_social.png\"/></a></span>&nbsp;&nbsp;&nbsp;<span><a href=\"https://devtron.ai/blog\" style=\"font-size: 13px; font-weight: 400; color: #000A14\">Blog</a></span>&nbsp;&nbsp;&nbsp;<span><a href=\"https://devtron.ai/\" style=\"font-size: 13px; font-weight: 400; color: #000A14\">Website</a></span></div><div style=\"margin-left: 250px;\">© Devtron Labs 2023</div></div></table>"}');


