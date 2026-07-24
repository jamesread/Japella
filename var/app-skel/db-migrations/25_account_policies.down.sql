ALTER TABLE `posts`
  DROP FOREIGN KEY `fk_posts_submitted_by`,
  DROP FOREIGN KEY `fk_posts_account_policy`,
  DROP COLUMN `submission_source`,
  DROP COLUMN `submitted_by_user_id`,
  DROP COLUMN `account_policy_id`,
  DROP COLUMN `approval_stage`;

DELETE rp FROM `rbac_role_permissions` rp
  INNER JOIN `rbac_permissions` p ON p.id = rp.permission_id
  WHERE p.name = 'account-policies.manage';
DELETE FROM `rbac_permissions` WHERE `name` = 'account-policies.manage';

DROP TABLE IF EXISTS `post_approvals`;
DROP TABLE IF EXISTS `account_policy_approval_stages`;
DROP TABLE IF EXISTS `account_policy_social_accounts`;
DROP TABLE IF EXISTS `account_policies`;
