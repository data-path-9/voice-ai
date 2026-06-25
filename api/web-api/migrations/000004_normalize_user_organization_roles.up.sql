UPDATE public.user_organization_roles
SET role = CASE
    WHEN role = 'OWNER' THEN 'owner'
    WHEN role = 'ADMIN' THEN 'admin'
    WHEN role = 'MEMBER' THEN 'member'
    WHEN role IN ('owner', 'admin', 'member') THEN role
    ELSE 'member'
END
WHERE role NOT IN ('owner', 'admin', 'member');
